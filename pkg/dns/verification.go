package dns

import (
	"fmt"
	"net"
	"strings"
	"time"

	mdns "github.com/miekg/dns"
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

// Public DNS resolvers to try (Termux-compatible)
var PublicDNSResolvers = []string{
	"8.8.8.8:53",    // Google DNS
	"1.1.1.1:53",    // Cloudflare DNS
	"8.8.4.4:53",    // Google DNS secondary
	"1.0.0.1:53",    // Cloudflare DNS secondary
}

// lookupNSRecords queries NS records using direct DNS queries
// This works in Termux and restricted environments where system resolver is unavailable
func lookupNSRecords(domain string) ([]string, error) {
	var lastErr error

	// Try each public DNS resolver
	for _, resolver := range PublicDNSResolvers {
		ns, err := queryNSRecords(domain, resolver, 5*time.Second)
		if err == nil && len(ns) > 0 {
			return ns, nil
		}
		lastErr = err
	}

	// If all direct queries failed, try system resolver as fallback
	nsRecords, err := net.LookupNS(domain)
	if err == nil {
		result := make([]string, len(nsRecords))
		for i, ns := range nsRecords {
			result[i] = ns.Host
		}
		return result, nil
	}

	// Return the last error from direct queries
	if lastErr != nil {
		return nil, fmt.Errorf("all DNS resolvers failed: %w", lastErr)
	}
	return nil, err
}

// queryNSRecords performs a direct DNS query for NS records
func queryNSRecords(domain, resolver string, timeout time.Duration) ([]string, error) {
	c := &mdns.Client{
		Timeout: timeout,
	}

	m := &mdns.Msg{}
	m.SetQuestion(mdns.Fqdn(domain), mdns.TypeNS)
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, resolver)
	if err != nil {
		return nil, fmt.Errorf("DNS query failed: %w", err)
	}

	if r.Rcode != mdns.RcodeSuccess {
		return nil, fmt.Errorf("DNS query returned %s", mdns.RcodeToString[r.Rcode])
	}

	var nameservers []string
	for _, ans := range r.Answer {
		if ns, ok := ans.(*mdns.NS); ok {
			nameservers = append(nameservers, ns.Ns)
		}
	}

	return nameservers, nil
}

// VerifyNSDelegation checks if a domain's NS records point to expected nameservers
// Returns a VerificationResult with detailed information about the delegation status
// Now uses direct DNS queries for Termux compatibility
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

	// Look up NS records using direct DNS queries
	nsRecords, err := lookupNSRecords(domain)
	if err != nil {
		result.Error = fmt.Errorf("DNS lookup failed for %s: %w", domain, err)
		return result
	}

	// Extract and normalize actual nameservers
	result.ActualNS = nsRecords

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
