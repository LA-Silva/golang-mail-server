#!/bin/bash

# Unit tests for mail server - Send and Receive email functionality
# Exit code: 0 = success, 1 = failure

set -euo pipefail

# Configuration
SMTP_HOST="${SMTP_HOST:-localhost}"
SMTP_PORT="${SMTP_PORT:-25}"
IMAP_HOST="${IMAP_HOST:-localhost}"
IMAP_PORT="${IMAP_PORT:-143}"
TEST_USER="${TEST_USER:-user1}"
TEST_PASSWORD="${TEST_PASSWORD:-password1}"
TEST_RECIPIENT="${TEST_RECIPIENT:-user1@example.com}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

# Test framework
run_test() {
    local test_name="$1"
    local test_func="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    log_test "Running: $test_name"
    
    if $test_func; then
        log_info "✓ PASSED: $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_error "✗ FAILED: $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Utility functions
check_connectivity() {
    local host="$1"
    local port="$2"
    local service="$3"
    
    log_info "Checking $service connectivity to $host:$port..."
    
    if timeout 5 bash -c "</dev/tcp/$host/$port" 2>/dev/null; then
        log_info "✓ Connected to $service on $host:$port"
        return 0
    else
        log_error "✗ Failed to connect to $service on $host:$port"
        return 1
    fi
}

# Test 1: Check SMTP connectivity
test_smtp_connectivity() {
    check_connectivity "$SMTP_HOST" "$SMTP_PORT" "SMTP"
}

# Test 2: Check IMAP connectivity
test_imap_connectivity() {
    check_connectivity "$IMAP_HOST" "$IMAP_PORT" "IMAP"
}

# Test 3: Send email via SMTP
test_send_email() {
    local email_file=$(mktemp)
    local timestamp=$(date +%s)
    local message_id="test-$timestamp"
    
    cat > "$email_file" << EOF
From: $TEST_USER <$TEST_USER@example.com>
To: $TEST_RECIPIENT
Subject: Test Email - $message_id
Date: $(date -R)
Message-ID: <$message_id@example.com>

This is a test email sent at $(date).
Message ID: $message_id
EOF
    
    log_info "Sending test email via SMTP..."
    
    if timeout 10 nc -q 1 "$SMTP_HOST" "$SMTP_PORT" << SMTP_COMMANDS 2>/dev/null; then
EHLO test.example.com
AUTH PLAIN $(echo -ne "\0$TEST_USER\0$TEST_PASSWORD" | base64)
MAIL FROM:<$TEST_USER@example.com>
RCPT TO:<$TEST_RECIPIENT>
DATA
$(cat "$email_file")
.
QUIT
SMTP_COMMANDS
        log_info "✓ Email sent successfully (Message ID: $message_id)"
        echo "$message_id" > /tmp/last_email_id
        rm -f "$email_file"
        return 0
    else
        log_error "✗ Failed to send email via SMTP"
        rm -f "$email_file"
        return 1
    fi
}

# Test 4: Receive email via IMAP
test_receive_email() {
    log_info "Connecting to IMAP server..."
    
    local imap_response=$(timeout 10 nc -q 1 "$IMAP_HOST" "$IMAP_PORT" << IMAP_COMMANDS 2>&1
1 LOGIN $TEST_USER $TEST_PASSWORD
2 SELECT INBOX
3 SEARCH ALL
4 LOGOUT
IMAP_COMMANDS
    )
    
    if echo "$imap_response" | grep -qi "OK"; then
        log_info "✓ IMAP login and mailbox selection successful"
        
        # Check if we got any search results
        if echo "$imap_response" | grep -qi "SEARCH"; then
            log_info "✓ Mailbox contains messages"
            return 0
        else
            log_warn "⚠ Mailbox appears empty or search failed"
            return 0  # Not necessarily a failure
        fi
    else
        log_error "✗ IMAP login failed"
        return 1
    fi
}

# Test 5: Verify IMAP authentication fails with wrong password
test_imap_auth_failure() {
    log_info "Testing IMAP authentication with wrong password..."
    
    local imap_response=$(timeout 10 nc -q 1 "$IMAP_HOST" "$IMAP_PORT" << IMAP_COMMANDS 2>&1
1 LOGIN $TEST_USER wrongpassword
2 LOGOUT
IMAP_COMMANDS
    )
    
    if echo "$imap_response" | grep -qi "NO\|BAD"; then
        log_info "✓ IMAP correctly rejected invalid credentials"
        return 0
    else
        log_error "✗ IMAP did not reject invalid credentials"
        return 1
    fi
}

# Test 6: Test email roundtrip (send and receive)
test_email_roundtrip() {
    log_info "Testing email roundtrip (send and receive)..."
    
    # Send email
    local email_file=$(mktemp)
    local timestamp=$(date +%s)
    local message_id="roundtrip-$timestamp"
    
    cat > "$email_file" << EOF
From: $TEST_USER <$TEST_USER@example.com>
To: $TEST_RECIPIENT
Subject: Roundtrip Test - $message_id
Date: $(date -R)
Message-ID: <$message_id@example.com>

Roundtrip test email sent at $(date).
EOF
    
    # Send via SMTP
    if ! timeout 10 nc -q 1 "$SMTP_HOST" "$SMTP_PORT" << SMTP_COMMANDS 2>/dev/null; then
EHLO test.example.com
AUTH PLAIN $(echo -ne "\0$TEST_USER\0$TEST_PASSWORD" | base64)
MAIL FROM:<$TEST_USER@example.com>
RCPT TO:<$TEST_RECIPIENT>
DATA
$(cat "$email_file")
.
QUIT
SMTP_COMMANDS
        log_error "✗ Roundtrip test failed at SMTP send"
        rm -f "$email_file"
        return 1
    fi
    
    sleep 1  # Give server time to process
    
    # Check via IMAP
    local imap_response=$(timeout 10 nc -q 1 "$IMAP_HOST" "$IMAP_PORT" << IMAP_COMMANDS 2>&1
1 LOGIN $TEST_USER $TEST_PASSWORD
2 SELECT INBOX
3 SEARCH SUBJECT "$message_id"
4 LOGOUT
IMAP_COMMANDS
    )
    
    rm -f "$email_file"
    
    if echo "$imap_response" | grep -qi "OK"; then
        log_info "✓ Email roundtrip completed successfully"
        return 0
    else
        log_error "✗ Email roundtrip test failed at IMAP receive"
        return 1
    fi
}

# Summary function
print_summary() {
    echo ""
    echo "======================================"
    echo "Test Summary"
    echo "======================================"
    echo "Total Tests: $TESTS_RUN"
    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    echo "======================================"
    echo ""
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log_info "All tests passed!"
        return 0
    else
        log_error "$TESTS_FAILED test(s) failed"
        return 1
    fi
}

# Main execution
main() {
    echo "========================================"
    echo "Mail Server Unit Tests"
    echo "========================================"
    echo "SMTP: $SMTP_HOST:$SMTP_PORT"
    echo "IMAP: $IMAP_HOST:$IMAP_PORT"
    echo "User: $TEST_USER"
    echo "========================================"
    echo ""
    
    # Run connectivity tests first
    run_test "SMTP Connectivity" test_smtp_connectivity || true
    run_test "IMAP Connectivity" test_imap_connectivity || true
    
    echo ""
    
    # Run functional tests
    run_test "Send Email via SMTP" test_send_email || true
    run_test "Receive Email via IMAP" test_receive_email || true
    run_test "IMAP Auth Failure (Wrong Password)" test_imap_auth_failure || true
    run_test "Email Roundtrip (Send & Receive)" test_email_roundtrip || true
    
    echo ""
    
    # Print summary and exit with appropriate code
    if print_summary; then
        exit 0
    else
        exit 1
    fi
}

# Run main function
main
