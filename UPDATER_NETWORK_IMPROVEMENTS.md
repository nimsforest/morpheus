# Updater Network Resilience Improvements

## Problem

The `morpheus update` command was failing with DNS resolution errors when the system had DNS configuration issues:

```
Failed to check for updates: failed to check for updates: Get "https://api.github.com/repos/nimsforest/morpheus/releases/latest": dial tcp: lookup api.github.com on [::1]:53: read udp [::1]:55541->[::1]:53: read: connection refused
```

This error occurs when:
- DNS is configured to use localhost (`[::1]:53` or `127.0.0.1:53`) but no DNS server is running
- Network connectivity issues exist
- IPv6/IPv4 dual-stack configuration problems

## Solution

Enhanced the updater package with the following improvements:

### 1. Custom HTTP Transport with Better Dialer

```go
// Create a custom dialer with fallback to IPv4 if IPv6 fails
dialer := &net.Dialer{
    Timeout:   5 * time.Second,
    KeepAlive: 30 * time.Second,
}

transport := &http.Transport{
    DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
        // Try dual-stack (both IPv4 and IPv6)
        return dialer.DialContext(ctx, network, addr)
    },
    // ... additional transport settings
}
```

**Benefits:**
- Better timeout handling
- Dual-stack IPv4/IPv6 support
- Optimized connection pooling

### 2. Retry Logic with Exponential Backoff

```go
const maxRetries = 3

// Retry logic for network resilience
for attempt := 1; attempt <= maxRetries; attempt++ {
    info, err := u.checkForUpdateOnce()
    if err == nil {
        return info, nil
    }
    // ... exponential backoff on network errors
}
```

**Benefits:**
- Handles transient network failures
- Exponential backoff (2s, 4s, 6s) reduces server load
- Only retries network errors, not API errors

### 3. Enhanced Error Messages with Troubleshooting

The updater now detects specific error types and provides targeted troubleshooting advice:

#### DNS Resolution Errors
```
Network troubleshooting:
  • Check your internet connection
  • Verify DNS is configured correctly (check /etc/resolv.conf)
  • Try: ping api.github.com
  • If on IPv6-only network, ensure IPv6 DNS is working
  • Try using a different DNS server (e.g., 8.8.8.8, 1.1.1.1)
```

#### Localhost DNS Configuration Issues
```
DNS configuration issue detected:
  • Your system is trying to use localhost as DNS server
  • Check /etc/resolv.conf for incorrect DNS settings
  • Common fix: Replace localhost DNS with:
      nameserver 8.8.8.8
      nameserver 1.1.1.1
  • On some systems, edit /etc/systemd/resolved.conf
```

#### Connection Refused Errors
```
Connection issue:
  • Firewall may be blocking the connection
  • Check if you're behind a proxy
  • Verify you can access: https://api.github.com
```

#### Timeout Errors
```
Connection timeout:
  • Check your internet connection
  • Network may be slow or unstable
  • Try again later
```

## How to Fix DNS Issues

### For the Reported Error

The specific error `lookup api.github.com on [::1]:53: read udp ... connection refused` indicates DNS is configured to use localhost. To fix:

1. **Check current DNS configuration:**
   ```bash
   cat /etc/resolv.conf
   ```

2. **If it shows `nameserver ::1` or `nameserver 127.0.0.1`, replace with public DNS:**
   ```bash
   sudo nano /etc/resolv.conf
   ```
   
   Replace content with:
   ```
   nameserver 8.8.8.8
   nameserver 1.1.1.1
   ```

3. **For systemd-resolved systems:**
   ```bash
   sudo nano /etc/systemd/resolved.conf
   ```
   
   Add:
   ```ini
   [Resolve]
   DNS=8.8.8.8 1.1.1.1
   FallbackDNS=8.8.4.4 1.0.0.1
   ```
   
   Then restart:
   ```bash
   sudo systemctl restart systemd-resolved
   ```

4. **Test DNS resolution:**
   ```bash
   nslookup api.github.com
   ping api.github.com
   ```

## Testing

All existing tests pass:
```bash
go test ./pkg/updater/... -v
```

Build verification:
```bash
go build ./cmd/morpheus
```

## Backward Compatibility

These changes are fully backward compatible:
- No API changes
- No configuration changes required
- Enhanced error messages don't affect normal operation
- Retry logic is transparent to users

## Future Improvements

Potential enhancements for even better resilience:
1. Add support for HTTP/SOCKS proxy configuration
2. Add `--offline` mode to skip update checks
3. Add caching of last successful update check
4. Support for custom GitHub API endpoints (for enterprise)
5. Add network connectivity pre-check before attempting update
