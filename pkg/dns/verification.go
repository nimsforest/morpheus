package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// dohLookupNS performs DNS over HTTPS lookup for NS records
func dohLookupNS(domain string) ([]*net.NS, error) {
	// Try Google DNS over HTTPS
	url := fmt.Sprintf("https://dns.google/resolve?name=%s&type=NS", domain)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Try Cloudflare DoH as fallback
		url = fmt.Sprintf("https://cloudflare-dns.com/dns-query?name=%s&type=NS", domain)
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/dns-json")

		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("DoH lookup failed: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DoH query returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Answer []struct {
			Data string `json:"data"`
		} `json:"Answer"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Answer) == 0 {
		return nil, fmt.Errorf("no NS records found")
	}

	var nsRecords []*net.NS
	for _, answer := range result.Answer {
		nsRecords = append(nsRecords, &net.NS{Host: answer.Data})
	}

	return nsRecords, nil
}

// lookupNSWithFallback attempts to lookup NS records using system resolver first,
// then falls back to public DNS servers if the system resolver fails
func lookupNSWithFallback(domain string) ([]*net.NS, error) {
	// Try system resolver first
	nsRecords, err := net.LookupNS(domain)
	if err == nil {
		return nsRecords, nil
	}

	// If system resolver fails with connection refused, try public DNS servers
	systemErr := err
	if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
		// Try with custom resolvers (Google DNS and Cloudflare DNS)
		for _, dnsServer := range []string{"8.8.8.8:53", "1.1.1.1:53"} {
			resolver := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{
						Timeout: 5 * time.Second,
					}
					return d.DialContext(ctx, network, dnsServer)
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			nsRecords, lookupErr := resolver.LookupNS(ctx, domain)
			cancel()

			if lookupErr == nil {
				return nsRecords, nil
			}
			// Update err to the latest error
			err = lookupErr
		}

		// If UDP DNS failed, try DNS over HTTPS as final fallback
		nsRecords, dohErr := dohLookupNS(domain)
		if dohErr == nil {
			return nsRecords, nil
		}

		// If all fallback attempts failed, return error with context
		return nil, fmt.Errorf("all DNS lookups failed (system resolver: %v, UDP DNS timeout, DoH: %v)", systemErr, dohErr)
	}

	return nil, err
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

	// Look up NS records for the domain (with fallback to public DNS)
	nsRecords, err := lookupNSWithFallback(domain)
	if err != nil {
		result.Error = fmt.Errorf("DNS lookup failed for %s: %w", domain, err)
		return result
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

// dohLookupMX performs DNS over HTTPS lookup for MX records
func dohLookupMX(domain string) ([]*net.MX, error) {
	// Try Google DNS over HTTPS
	url := fmt.Sprintf("https://dns.google/resolve?name=%s&type=MX", domain)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Try Cloudflare DoH as fallback
		url = fmt.Sprintf("https://cloudflare-dns.com/dns-query?name=%s&type=MX", domain)
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/dns-json")

		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("DoH lookup failed: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DoH query returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Answer []struct {
			Data string `json:"data"`
		} `json:"Answer"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Answer) == 0 {
		return nil, fmt.Errorf("no MX records found")
	}

	var mxRecords []*net.MX
	for _, answer := range result.Answer {
		// MX data format: "priority host"
		var pref uint16
		var host string
		fmt.Sscanf(answer.Data, "%d %s", &pref, &host)
		mxRecords = append(mxRecords, &net.MX{
			Host: host,
			Pref: pref,
		})
	}

	return mxRecords, nil
}

// lookupMXWithFallback attempts to lookup MX records using system resolver first,
// then falls back to DNS over HTTPS if the system resolver fails
func lookupMXWithFallback(domain string) ([]*net.MX, error) {
	// Try system resolver first
	mxRecords, err := net.LookupMX(domain)
	if err == nil {
		return mxRecords, nil
	}

	// If system resolver fails, try DNS over HTTPS
	if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
		mxRecords, dohErr := dohLookupMX(domain)
		if dohErr == nil {
			return mxRecords, nil
		}
		return nil, fmt.Errorf("MX lookup failed (system: %v, DoH: %v)", err, dohErr)
	}

	return nil, err
}

// dohLookupTXT performs DNS over HTTPS lookup for TXT records
func dohLookupTXT(domain string) ([]string, error) {
	// Try Google DNS over HTTPS
	url := fmt.Sprintf("https://dns.google/resolve?name=%s&type=TXT", domain)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Try Cloudflare DoH as fallback
		url = fmt.Sprintf("https://cloudflare-dns.com/dns-query?name=%s&type=TXT", domain)
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/dns-json")

		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("DoH lookup failed: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DoH query returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Answer []struct {
			Data string `json:"data"`
		} `json:"Answer"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Answer) == 0 {
		return nil, fmt.Errorf("no TXT records found")
	}

	var txtRecords []string
	for _, answer := range result.Answer {
		// Remove quotes from TXT data if present
		data := strings.Trim(answer.Data, "\"")
		txtRecords = append(txtRecords, data)
	}

	return txtRecords, nil
}

// lookupTXTWithFallback attempts to lookup TXT records using system resolver first,
// then falls back to DNS over HTTPS if the system resolver fails
func lookupTXTWithFallback(domain string) ([]string, error) {
	// Try system resolver first
	txtRecords, err := net.LookupTXT(domain)
	if err == nil {
		return txtRecords, nil
	}

	// If system resolver fails, try DNS over HTTPS
	if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
		txtRecords, dohErr := dohLookupTXT(domain)
		if dohErr == nil {
			return txtRecords, nil
		}
		return nil, fmt.Errorf("TXT lookup failed (system: %v, DoH: %v)", err, dohErr)
	}

	return nil, err
}

// EmailVerificationResult contains the result of email DNS verification
type EmailVerificationResult struct {
	Domain       string
	HasMX        bool
	MXRecords    []*net.MX
	HasSPF       bool
	SPFRecord    string
	HasDMARC     bool
	DMARCRecord  string
	HasDKIM      bool
	DKIMSelector string
	Error        error
}

// VerifyEmailDNS checks if a domain has proper email DNS records (MX, SPF, DMARC)
func VerifyEmailDNS(domain string) *EmailVerificationResult {
	result := &EmailVerificationResult{
		Domain: domain,
	}

	// Check MX records
	mxRecords, err := lookupMXWithFallback(domain)
	if err == nil && len(mxRecords) > 0 {
		result.HasMX = true
		result.MXRecords = mxRecords
	}

	// Check SPF (TXT record at apex)
	txtRecords, err := lookupTXTWithFallback(domain)
	if err == nil {
		for _, txt := range txtRecords {
			if strings.HasPrefix(txt, "v=spf1") {
				result.HasSPF = true
				result.SPFRecord = txt
				break
			}
		}
	}

	// Check DMARC (TXT record at _dmarc subdomain)
	dmarcRecords, err := lookupTXTWithFallback("_dmarc." + domain)
	if err == nil && len(dmarcRecords) > 0 {
		for _, txt := range dmarcRecords {
			if strings.HasPrefix(txt, "v=DMARC1") {
				result.HasDMARC = true
				result.DMARCRecord = txt
				break
			}
		}
	}

	return result
}
