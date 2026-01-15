package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// VerificationResult contains the result of a DNS delegation verification
type VerificationResult struct {
	Domain       string   // The domain that was verified
	Delegated    bool     // Whether NS delegation is correct
	ExpectedNS   []string // Expected nameservers
	ActualNS     []string // Actual nameservers found
	MatchingNS   []string // Nameservers that match expected
	MissingNS    []string // Expected nameservers not found
	ExtraNS      []string // Actual nameservers not in expected list
	Error        error    // Any error that occurred during lookup
	PartialMatch bool     // True if some but not all NS records match
}

// createCustomResolver creates a DNS resolver that uses public DNS servers
// instead of the system's default resolver. This avoids issues where the
// system DNS points to localhost (::1:53) but no DNS server is running.
func createCustomResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 10 * time.Second}
			// Try Google DNS first
			conn, err := d.DialContext(ctx, "udp", "8.8.8.8:53")
			if err != nil {
				// Fallback to Cloudflare DNS
				conn, err = d.DialContext(ctx, "udp", "1.1.1.1:53")
			}
			if err != nil {
				// Last fallback to Quad9
				conn, err = d.DialContext(ctx, "udp", "9.9.9.9:53")
			}
			return conn, err
		},
	}
}

// dohResponse represents a DNS-over-HTTPS response from Google or Cloudflare
type dohResponse struct {
	Status   int  `json:"Status"`
	Answer   []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// lookupNSviaDoH performs DNS NS lookup using DNS-over-HTTPS
// This works in environments where UDP port 53 is filtered but HTTPS works
func lookupNSviaDoH(ctx context.Context, domain string) ([]*net.NS, error) {
	// Try Google DNS-over-HTTPS first
	urls := []string{
		fmt.Sprintf("https://dns.google/resolve?name=%s&type=NS", domain),
		fmt.Sprintf("https://cloudflare-dns.com/dns-query?name=%s&type=NS", domain),
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var lastErr error
	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Accept", "application/dns-json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("DoH server returned status %d", resp.StatusCode)
			continue
		}

		var dohResp dohResponse
		if err := json.NewDecoder(resp.Body).Decode(&dohResp); err != nil {
			lastErr = err
			continue
		}

		if dohResp.Status != 0 {
			lastErr = fmt.Errorf("DNS query failed with status %d", dohResp.Status)
			continue
		}

		// Convert to net.NS format
		var nsRecords []*net.NS
		for _, answer := range dohResp.Answer {
			if answer.Type == 2 { // NS record type
				nsRecords = append(nsRecords, &net.NS{Host: answer.Data})
			}
		}

		if len(nsRecords) > 0 {
			return nsRecords, nil
		}

		lastErr = fmt.Errorf("no NS records found")
	}

	return nil, fmt.Errorf("DoH lookup failed: %w", lastErr)
}

// VerifyNSDelegation checks if a domain's NS records point to expected nameservers
// Returns a VerificationResult with detailed information about the delegation status
func VerifyNSDelegation(domain string, expectedNS []string) *VerificationResult {
	result := &VerificationResult{
		Domain:     domain,
		ExpectedNS: expectedNS,
	}

	// Normalize expected nameservers (lowercase, remove trailing dots)
	normalizedExpected := make(map[string]bool)
	for _, ns := range expectedNS {
		normalizedExpected[NormalizeNS(ns)] = true
	}

	var nsRecords []*net.NS
	var err error

	// Try system resolver first (works in most environments) - quick timeout
	systemCtx, systemCancel := context.WithTimeout(context.Background(), 3*time.Second)
	done := make(chan bool, 1)
	go func() {
		nsRecords, err = net.LookupNS(domain)
		done <- true
	}()
	select {
	case <-done:
		systemCancel()
		if err == nil {
			// Success with system resolver
		} else {
			// System resolver failed, try UDP resolver
			udpCtx, udpCancel := context.WithTimeout(context.Background(), 5*time.Second)
			resolver := createCustomResolver()
			nsRecords, err = resolver.LookupNS(udpCtx, domain)
			udpCancel()

			// If UDP DNS fails, fall back to DNS-over-HTTPS
			if err != nil {
				dohCtx, dohCancel := context.WithTimeout(context.Background(), 15*time.Second)
				nsRecords, err = lookupNSviaDoH(dohCtx, domain)
				dohCancel()
				if err != nil {
					result.Error = fmt.Errorf("DNS lookup failed for %s: %w", domain, err)
					return result
				}
			}
		}
	case <-systemCtx.Done():
		systemCancel()
		// Timeout, try UDP resolver
		udpCtx, udpCancel := context.WithTimeout(context.Background(), 5*time.Second)
		resolver := createCustomResolver()
		nsRecords, err = resolver.LookupNS(udpCtx, domain)
		udpCancel()

		// If UDP DNS fails, fall back to DNS-over-HTTPS
		if err != nil {
			dohCtx, dohCancel := context.WithTimeout(context.Background(), 15*time.Second)
			nsRecords, err = lookupNSviaDoH(dohCtx, domain)
			dohCancel()
			if err != nil {
				result.Error = fmt.Errorf("DNS lookup failed for %s: %w", domain, err)
				return result
			}
		}
	}

	// Extract and normalize actual nameservers
	for _, ns := range nsRecords {
		result.ActualNS = append(result.ActualNS, ns.Host)
	}

	// Build normalized actual NS set
	normalizedActual := make(map[string]bool)
	for _, ns := range result.ActualNS {
		normalizedActual[NormalizeNS(ns)] = true
	}

	// Calculate matching, missing, and extra nameservers
	for ns := range normalizedExpected {
		if normalizedActual[ns] {
			result.MatchingNS = append(result.MatchingNS, ns)
		} else {
			result.MissingNS = append(result.MissingNS, ns)
		}
	}

	for ns := range normalizedActual {
		if !normalizedExpected[ns] {
			result.ExtraNS = append(result.ExtraNS, ns)
		}
	}

	// Determine delegation status
	// Consider delegated if at least one expected NS is present
	result.Delegated = len(result.MatchingNS) > 0 && len(result.MissingNS) == 0
	result.PartialMatch = len(result.MatchingNS) > 0 && len(result.MissingNS) > 0

	return result
}

// NormalizeNS normalizes a nameserver string by lowercasing and removing trailing dots
func NormalizeNS(ns string) string {
	ns = strings.ToLower(strings.TrimSpace(ns))
	return strings.TrimSuffix(ns, ".")
}

// CheckNSPropagation performs multiple DNS lookups to check if NS delegation has propagated
// It uses system DNS resolver, which may use caching
func CheckNSPropagation(domain string, expectedNS []string) (bool, error) {
	result := VerifyNSDelegation(domain, expectedNS)
	if result.Error != nil {
		return false, result.Error
	}
	return result.Delegated, nil
}

// FormatVerificationResult returns a human-readable string describing the verification result
func FormatVerificationResult(result *VerificationResult) string {
	var sb strings.Builder

	if result.Error != nil {
		sb.WriteString(fmt.Sprintf("Error verifying %s: %s\n", result.Domain, result.Error))
		return sb.String()
	}

	if result.Delegated {
		sb.WriteString(fmt.Sprintf("Domain %s is correctly delegated\n", result.Domain))
		sb.WriteString(fmt.Sprintf("  Nameservers: %s\n", strings.Join(result.ActualNS, ", ")))
	} else if result.PartialMatch {
		sb.WriteString(fmt.Sprintf("Domain %s has partial NS delegation\n", result.Domain))
		sb.WriteString(fmt.Sprintf("  Matching:  %s\n", strings.Join(result.MatchingNS, ", ")))
		sb.WriteString(fmt.Sprintf("  Missing:   %s\n", strings.Join(result.MissingNS, ", ")))
		if len(result.ExtraNS) > 0 {
			sb.WriteString(fmt.Sprintf("  Extra:     %s\n", strings.Join(result.ExtraNS, ", ")))
		}
	} else {
		sb.WriteString(fmt.Sprintf("Domain %s NS delegation NOT configured for expected nameservers\n", result.Domain))
		sb.WriteString(fmt.Sprintf("  Expected: %s\n", strings.Join(result.ExpectedNS, ", ")))
		sb.WriteString(fmt.Sprintf("  Actual:   %s\n", strings.Join(result.ActualNS, ", ")))
	}

	return sb.String()
}
