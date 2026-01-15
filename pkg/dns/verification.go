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

// dohResponse represents the JSON response from DNS-over-HTTPS providers
type dohResponse struct {
	Status   int  `json:"Status"`
	Answer   []dohAnswer `json:"Answer"`
}

type dohAnswer struct {
	Name string `json:"name"`
	Type int    `json:"type"`
	Data string `json:"data"`
}

// lookupNSviaDoH performs NS lookup using DNS-over-HTTPS
// This works even when UDP port 53 is blocked (e.g., in containers)
func lookupNSviaDoH(ctx context.Context, domain string) ([]*net.NS, error) {
	// Try multiple DoH providers
	providers := []string{
		"https://dns.google/resolve?name=" + domain + "&type=NS",
		"https://cloudflare-dns.com/dns-query?name=" + domain + "&type=NS",
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var lastErr error
	for _, provider := range providers {
		req, err := http.NewRequestWithContext(ctx, "GET", provider, nil)
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
			lastErr = fmt.Errorf("DoH provider returned status %d", resp.StatusCode)
			continue
		}

		var dohResp dohResponse
		if err := json.NewDecoder(resp.Body).Decode(&dohResp); err != nil {
			lastErr = err
			continue
		}

		if dohResp.Status != 0 {
			lastErr = fmt.Errorf("DoH response status: %d", dohResp.Status)
			continue
		}

		// Extract NS records (type 2 is NS)
		var nsRecords []*net.NS
		for _, answer := range dohResp.Answer {
			if answer.Type == 2 { // NS record type
				nsRecords = append(nsRecords, &net.NS{Host: answer.Data})
			}
		}

		if len(nsRecords) > 0 {
			return nsRecords, nil
		}

		lastErr = fmt.Errorf("no NS records found in DoH response")
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all DoH providers failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no DoH providers available")
}

// VerifyNSDelegation checks if a domain's NS records point to expected nameservers
// Returns a VerificationResult with detailed information about the delegation status
// Uses a 3-tier fallback system: system resolver → custom UDP resolver → DNS-over-HTTPS
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

	// 3-tier fallback system for DNS lookups
	var nsRecords []*net.NS
	var err error

	// Tier 1: Try system resolver with 3s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resolver := &net.Resolver{}
	nsRecords, err = resolver.LookupNS(ctx, domain)
	if err == nil && len(nsRecords) > 0 {
		// System resolver succeeded
		goto processRecords
	}

	// Tier 2: Try custom UDP resolver (8.8.8.8, 1.1.1.1) with 5s timeout
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, dnsServer := range []string{"8.8.8.8:53", "1.1.1.1:53"} {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 5 * time.Second,
				}
				return d.DialContext(ctx, "udp", dnsServer)
			},
		}
		nsRecords, err = resolver.LookupNS(ctx, domain)
		if err == nil && len(nsRecords) > 0 {
			// Custom UDP resolver succeeded
			goto processRecords
		}
	}

	// Tier 3: Fall back to DNS-over-HTTPS with 15s timeout
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	nsRecords, err = lookupNSviaDoH(ctx, domain)
	if err != nil {
		result.Error = fmt.Errorf("all DNS lookup methods failed for %s: %w", domain, err)
		return result
	}

processRecords:

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
