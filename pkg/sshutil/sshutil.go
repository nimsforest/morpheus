// Package sshutil provides utility functions for SSH-related formatting.
package sshutil

import (
	"fmt"
	"strings"
)

// FormatSSHCommand returns a properly formatted SSH command for display to users.
// IPv6 addresses do NOT need brackets for the ssh command.
// Example: ssh root@2001:db8::1
func FormatSSHCommand(user, ip string) string {
	return fmt.Sprintf("ssh %s@%s", user, ip)
}

// FormatSSHAddress returns a properly formatted address for TCP connections.
// IPv6 addresses need brackets when combined with a port.
// Example: [2001:db8::1]:22
func FormatSSHAddress(ip string, port int) string {
	// Check if this looks like an IPv6 address (contains colons and no brackets already)
	if strings.Contains(ip, ":") && !strings.HasPrefix(ip, "[") {
		return fmt.Sprintf("[%s]:%d", ip, port)
	}
	// IPv4 or already bracketed
	return fmt.Sprintf("%s:%d", ip, port)
}

// IsIPv6 returns true if the given IP address appears to be IPv6.
func IsIPv6(ip string) bool {
	return strings.Contains(ip, ":")
}
