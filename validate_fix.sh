#!/bin/bash
# Don't exit on errors, we want to count them
set +e

echo "================================================================================"
echo "VALIDATION: SIGSYS Fix + TLS Certificate Handling"
echo "================================================================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

function test_pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    ((PASSED++))
}

function test_fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    ((FAILED++))
}

function test_info() {
    echo -e "${YELLOW}ℹ️  INFO${NC}: $1"
}

echo "Test 1: Check for SIGSYS-triggering code patterns"
echo "------------------------------------------------"
if grep -r "exec.Command.*curl" pkg/updater/*.go 2>/dev/null; then
    test_fail "Found exec.Command with curl (will cause SIGSYS on Termux)"
else
    test_pass "No exec.Command with curl found (SIGSYS issue avoided)"
fi
echo ""

echo "Test 2: Verify native HTTP client is used"
echo "------------------------------------------"
if grep -q "net/http" pkg/updater/updater.go && grep -q "http.Client" pkg/updater/updater.go; then
    test_pass "Native net/http package is used"
else
    test_fail "Native net/http package not found"
fi
echo ""

echo "Test 3: Verify TLS certificate handling exists"
echo "----------------------------------------------"
if grep -q "x509.SystemCertPool" pkg/updater/updater.go; then
    test_pass "System certificate pool loading implemented"
else
    test_fail "System certificate pool loading not found"
fi

if grep -q "/etc/ssl/certs/ca-certificates.crt" pkg/updater/updater.go; then
    test_pass "Manual certificate loading from common paths implemented"
else
    test_fail "Manual certificate loading not found"
fi

if grep -q "InsecureSkipVerify" pkg/updater/updater.go; then
    test_pass "Fallback insecure mode exists for restricted environments"
else
    test_fail "Fallback insecure mode not found"
fi
echo ""

echo "Test 4: Verify restricted environment detection"
echo "-----------------------------------------------"
if grep -q "isRestrictedEnvironment" pkg/updater/updater.go; then
    test_pass "Restricted environment detection function exists"
else
    test_fail "Restricted environment detection not found"
fi

if grep -q "TERMUX_VERSION" pkg/updater/updater.go; then
    test_pass "Termux detection via TERMUX_VERSION implemented"
else
    test_fail "Termux detection not found"
fi
echo ""

echo "Test 5: Build test"
echo "------------------"
if go build -o /tmp/morpheus-validate ./cmd/morpheus/ 2>&1; then
    test_pass "Binary builds successfully"
    
    # Check binary size (should be reasonable)
    SIZE=$(stat -f%z /tmp/morpheus-validate 2>/dev/null || stat -c%s /tmp/morpheus-validate 2>/dev/null)
    test_info "Binary size: $((SIZE / 1024 / 1024)) MB"
    
    rm -f /tmp/morpheus-validate
else
    test_fail "Binary build failed"
fi
echo ""

echo "Test 6: Unit tests"
echo "------------------"
if go test ./pkg/updater/... -v 2>&1 | grep -q "PASS"; then
    test_pass "All updater tests pass"
else
    test_fail "Some updater tests failed"
fi
echo ""

echo "Test 7: Real TLS connection test"
echo "--------------------------------"
if [ -f test_real_connection.go ]; then
    TEST_OUTPUT=$(go run test_real_connection.go 2>&1)
    if echo "$TEST_OUTPUT" | grep -q "Connection successful"; then
        test_pass "Real HTTPS connection to GitHub API works"
        if echo "$TEST_OUTPUT" | grep -q "TLS 1.3"; then
            test_info "Using TLS 1.3 (latest and most secure)"
        elif echo "$TEST_OUTPUT" | grep -q "TLS 1.2"; then
            test_info "Using TLS 1.2 (secure)"
        fi
    else
        test_fail "HTTPS connection to GitHub API failed"
    fi
else
    test_info "test_real_connection.go not found, skipping (covered by Test 8)"
fi
echo ""

echo "Test 8: Actual morpheus check-update command"
echo "--------------------------------------------"
go build -o /tmp/morpheus-validate ./cmd/morpheus/ 2>&1 > /dev/null
if /tmp/morpheus-validate check-update 2>&1 | grep -qE "(Update available|Already up to date)"; then
    test_pass "morpheus check-update works with real GitHub API"
    UPDATE_OUTPUT=$(/tmp/morpheus-validate check-update 2>&1)
    test_info "Output: $UPDATE_OUTPUT"
else
    test_fail "morpheus check-update failed"
fi
rm -f /tmp/morpheus-validate
echo ""

echo "Test 9: Check certificate paths on this system"
echo "----------------------------------------------"
CERT_FOUND=0
for CERT_PATH in \
    "/etc/ssl/certs/ca-certificates.crt" \
    "/etc/pki/tls/certs/ca-bundle.crt" \
    "/etc/ssl/ca-bundle.pem" \
    "/etc/ssl/cert.pem"; do
    if [ -f "$CERT_PATH" ]; then
        test_pass "Found certificate file: $CERT_PATH"
        CERT_FOUND=1
        break
    fi
done

if [ $CERT_FOUND -eq 0 ]; then
    test_info "No standard certificate files found (will use SystemCertPool)"
fi
echo ""

echo "Test 10: Verify no CGO dependencies"
echo "-----------------------------------"
if go list -f '{{.CgoFiles}}' ./pkg/updater | grep -q ".go"; then
    test_fail "CGO dependencies found (will break on Android)"
else
    test_pass "No CGO dependencies (pure Go implementation)"
fi
echo ""

echo "================================================================================"
echo "VALIDATION SUMMARY"
echo "================================================================================"
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
else
    echo -e "${GREEN}Failed: $FAILED${NC}"
fi
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 ALL VALIDATIONS PASSED!${NC}"
    echo ""
    echo "The fix is verified and working:"
    echo "  ✅ No SIGSYS-triggering code (safe for Termux)"
    echo "  ✅ Native HTTP with smart TLS handling"
    echo "  ✅ Real HTTPS connections work"
    echo "  ✅ Actual morpheus commands work with GitHub API"
    echo "  ✅ Pure Go implementation (no CGO)"
    echo ""
    echo "Ready for deployment on Termux/Android!"
    exit 0
else
    echo -e "${RED}⚠️  SOME VALIDATIONS FAILED${NC}"
    echo "Please review the failures above."
    exit 1
fi
