package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a Proxmox VE API client
type Client struct {
	baseURL    string
	tokenID    string
	tokenValue string
	httpClient *http.Client
	node       string
}

// NewClient creates a new Proxmox API client
func NewClient(config ProviderConfig) (*Client, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("proxmox host is required")
	}
	if config.APITokenID == "" || config.APITokenSecret == "" {
		return nil, fmt.Errorf("proxmox API token credentials are required")
	}

	port := config.Port
	if port == 0 {
		port = 8006
	}

	baseURL := fmt.Sprintf("https://%s:%d/api2/json", config.Host, port)

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Configure TLS (self-signed certs are common in home labs)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !config.VerifySSL,
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	node := config.Node
	if node == "" {
		node = "pve"
	}

	return &Client{
		baseURL:    baseURL,
		tokenID:    config.APITokenID,
		tokenValue: config.APITokenSecret,
		httpClient: httpClient,
		node:       node,
	}, nil
}

// apiResponse wraps Proxmox API responses
type apiResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors interface{}     `json:"errors,omitempty"`
}

// request performs an HTTP request to the Proxmox API
func (c *Client) request(ctx context.Context, method, path string, body url.Values) (json.RawMessage, error) {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(body.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// API token authentication
	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.tokenID, c.tokenValue))

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return apiResp.Data, nil
}

// ListVMs returns all VMs on the configured node
func (c *Client) ListVMs(ctx context.Context) ([]*VM, error) {
	path := fmt.Sprintf("/nodes/%s/qemu", c.node)

	data, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var vms []*VM
	if err := json.Unmarshal(data, &vms); err != nil {
		return nil, fmt.Errorf("parse VMs: %w", err)
	}

	// Set node for all VMs
	for _, vm := range vms {
		vm.Node = c.node
	}

	return vms, nil
}

// GetVM returns a specific VM by VMID
func (c *Client) GetVM(ctx context.Context, vmid int) (*VM, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/current", c.node, vmid)

	data, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var vm VM
	if err := json.Unmarshal(data, &vm); err != nil {
		return nil, fmt.Errorf("parse VM: %w", err)
	}

	vm.VMID = vmid
	vm.Node = c.node

	return &vm, nil
}

// GetVMConfig returns the full configuration of a VM
func (c *Client) GetVMConfig(ctx context.Context, vmid int) (*VMConfig, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/config", c.node, vmid)

	data, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	// Parse raw config first to extract hostpci devices
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	var config VMConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Extract hostpci devices
	for key, val := range rawConfig {
		if strings.HasPrefix(key, "hostpci") {
			if strVal, ok := val.(string); ok {
				config.HostPCI = append(config.HostPCI, strVal)
			}
		}
	}

	return &config, nil
}

// StartVM starts a stopped VM
func (c *Client) StartVM(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/start", c.node, vmid)

	data, err := c.request(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", err
	}

	// Response is the task UPID
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("parse UPID: %w", err)
	}

	return upid, nil
}

// StopVM stops a running VM (immediate, like pulling the power)
func (c *Client) StopVM(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", c.node, vmid)

	data, err := c.request(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("parse UPID: %w", err)
	}

	return upid, nil
}

// ShutdownVM gracefully shuts down a VM (ACPI shutdown)
func (c *Client) ShutdownVM(ctx context.Context, vmid int, timeout int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/shutdown", c.node, vmid)

	params := url.Values{}
	if timeout > 0 {
		params.Set("timeout", fmt.Sprintf("%d", timeout))
	}
	params.Set("forceStop", "1") // Force stop after timeout

	data, err := c.request(ctx, http.MethodPost, path, params)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("parse UPID: %w", err)
	}

	return upid, nil
}

// GetTaskStatus returns the status of an async task
func (c *Client) GetTaskStatus(ctx context.Context, upid string) (*TaskStatus, error) {
	// Extract node from UPID (format: UPID:node:...)
	parts := strings.Split(upid, ":")
	node := c.node
	if len(parts) >= 2 {
		node = parts[1]
	}

	path := fmt.Sprintf("/nodes/%s/tasks/%s/status", node, url.PathEscape(upid))

	data, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var status TaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("parse task status: %w", err)
	}

	status.UPID = upid
	return &status, nil
}

// WaitForTask waits for an async task to complete
func (c *Client) WaitForTask(ctx context.Context, upid string, pollInterval time.Duration) (*TaskStatus, error) {
	if pollInterval == 0 {
		pollInterval = time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := c.GetTaskStatus(ctx, upid)
			if err != nil {
				return nil, err
			}
			if !status.IsRunning() {
				return status, nil
			}
		}
	}
}

// WaitForVMStatus waits for a VM to reach a specific status
func (c *Client) WaitForVMStatus(ctx context.Context, vmid int, targetStatus VMStatus, pollInterval time.Duration) error {
	if pollInterval == 0 {
		pollInterval = time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			vm, err := c.GetVM(ctx, vmid)
			if err != nil {
				return err
			}
			if vm.Status == targetStatus {
				return nil
			}
		}
	}
}

// GetVMIPs returns IP addresses of a VM (requires QEMU guest agent)
func (c *Client) GetVMIPs(ctx context.Context, vmid int) ([]string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/agent/network-get-interfaces", c.node, vmid)

	data, err := c.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		// Guest agent may not be running
		return nil, nil
	}

	var result struct {
		Result []struct {
			Name        string `json:"name"`
			IPAddresses []struct {
				IPAddress string `json:"ip-address"`
				IPType    string `json:"ip-address-type"`
			} `json:"ip-addresses"`
		} `json:"result"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, nil // Don't fail if we can't parse
	}

	var ips []string
	for _, iface := range result.Result {
		// Skip loopback
		if iface.Name == "lo" {
			continue
		}
		for _, addr := range iface.IPAddresses {
			// Prefer IPv4 for display
			if addr.IPType == "ipv4" && !strings.HasPrefix(addr.IPAddress, "127.") {
				ips = append(ips, addr.IPAddress)
			}
		}
	}

	return ips, nil
}

// GetNodes returns all nodes in the cluster
func (c *Client) GetNodes(ctx context.Context) ([]*Node, error) {
	data, err := c.request(ctx, http.MethodGet, "/nodes", nil)
	if err != nil {
		return nil, err
	}

	var nodes []*Node
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("parse nodes: %w", err)
	}

	return nodes, nil
}

// Ping checks if the Proxmox API is reachable
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.GetNodes(ctx)
	return err
}
