package dns

import (
	"fmt"
	"net"
	"strings"
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

	// Look up NS records for the domain
	nsRecords, err := net.LookupNS(domain)
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

// MXRecord represents an MX record with priority and mail server
type MXRecord struct {
	Priority int
	Server   string
}

// MXVerificationResult contains the result of an MX record verification
type MXVerificationResult struct {
	Domain       string     // The domain that was verified
	Configured   bool       // Whether MX records are correctly configured
	ExpectedMX   []MXRecord // Expected MX records
	ActualMX     []MXRecord // Actual MX records found
	MatchingMX   []MXRecord // MX records that match expected
	MissingMX    []MXRecord // Expected MX records not found
	ExtraMX      []MXRecord // Actual MX records not in expected list
	Error        error      // Any error that occurred during lookup
	PartialMatch bool       // True if some but not all MX records match
}

// VerifyMXRecords checks if a domain's MX records match expected values
// Returns an MXVerificationResult with detailed information about the MX configuration
func VerifyMXRecords(domain string, expectedMX []MXRecord) *MXVerificationResult {
	result := &MXVerificationResult{
		Domain:     domain,
		ExpectedMX: expectedMX,
	}

	// Look up MX records for the domain
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		result.Error = fmt.Errorf("MX lookup failed for %s: %w", domain, err)
		return result
	}

	// Convert actual MX records to our format
	for _, mx := range mxRecords {
		result.ActualMX = append(result.ActualMX, MXRecord{
			Priority: int(mx.Pref),
			Server:   strings.TrimSuffix(strings.ToUpper(mx.Host), "."),
		})
	}

	// Build expected MX map (normalized server -> priority)
	expectedMap := make(map[string]int)
	for _, mx := range expectedMX {
		server := strings.ToUpper(strings.TrimSuffix(mx.Server, "."))
		expectedMap[server] = mx.Priority
	}

	// Build actual MX map (normalized server -> priority)
	actualMap := make(map[string]int)
	for _, mx := range result.ActualMX {
		server := strings.ToUpper(strings.TrimSuffix(mx.Server, "."))
		actualMap[server] = mx.Priority
	}

	// Calculate matching, missing, and extra MX records
	for server, priority := range expectedMap {
		if actualPriority, exists := actualMap[server]; exists && actualPriority == priority {
			result.MatchingMX = append(result.MatchingMX, MXRecord{Priority: priority, Server: server})
		} else {
			result.MissingMX = append(result.MissingMX, MXRecord{Priority: priority, Server: server})
		}
	}

	for server, priority := range actualMap {
		if expectedPriority, exists := expectedMap[server]; !exists || expectedPriority != priority {
			result.ExtraMX = append(result.ExtraMX, MXRecord{Priority: priority, Server: server})
		}
	}

	// Determine configuration status
	// Consider configured if all expected MX records are present with correct priorities
	result.Configured = len(result.MatchingMX) == len(expectedMX) && len(result.MissingMX) == 0
	result.PartialMatch = len(result.MatchingMX) > 0 && len(result.MissingMX) > 0

	return result
}

// FormatMXVerificationResult returns a human-readable string describing the MX verification result
func FormatMXVerificationResult(result *MXVerificationResult) string {
	var sb strings.Builder

	if result.Error != nil {
		sb.WriteString(fmt.Sprintf("Error verifying MX records for %s: %s\n", result.Domain, result.Error))
		return sb.String()
	}

	if result.Configured {
		sb.WriteString(fmt.Sprintf("Domain %s has correct MX configuration\n", result.Domain))
		sb.WriteString("  MX records:\n")
		for _, mx := range result.ActualMX {
			sb.WriteString(fmt.Sprintf("    %d %s\n", mx.Priority, mx.Server))
		}
	} else if result.PartialMatch {
		sb.WriteString(fmt.Sprintf("Domain %s has partial MX configuration\n", result.Domain))
		if len(result.MatchingMX) > 0 {
			sb.WriteString("  Matching:\n")
			for _, mx := range result.MatchingMX {
				sb.WriteString(fmt.Sprintf("    %d %s\n", mx.Priority, mx.Server))
			}
		}
		if len(result.MissingMX) > 0 {
			sb.WriteString("  Missing:\n")
			for _, mx := range result.MissingMX {
				sb.WriteString(fmt.Sprintf("    %d %s\n", mx.Priority, mx.Server))
			}
		}
		if len(result.ExtraMX) > 0 {
			sb.WriteString("  Extra:\n")
			for _, mx := range result.ExtraMX {
				sb.WriteString(fmt.Sprintf("    %d %s\n", mx.Priority, mx.Server))
			}
		}
	} else {
		sb.WriteString(fmt.Sprintf("Domain %s MX records NOT configured as expected\n", result.Domain))
		sb.WriteString("  Expected:\n")
		for _, mx := range result.ExpectedMX {
			sb.WriteString(fmt.Sprintf("    %d %s\n", mx.Priority, mx.Server))
		}
		sb.WriteString("  Actual:\n")
		if len(result.ActualMX) == 0 {
			sb.WriteString("    (none)\n")
		} else {
			for _, mx := range result.ActualMX {
				sb.WriteString(fmt.Sprintf("    %d %s\n", mx.Priority, mx.Server))
			}
		}
	}

	return sb.String()
}
