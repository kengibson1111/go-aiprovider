package utils

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// NetworkMonitorConfig configures the network monitor
type NetworkMonitorConfig struct {
	CheckInterval time.Duration
	Timeout       time.Duration
	MaxTimeout    time.Duration
	TestEndpoints []string
}

// NetworkStatus represents the current network connectivity state
type NetworkStatus int

const (
	NetworkStatusUnknown NetworkStatus = iota
	NetworkStatusOnline
	NetworkStatusOffline
	NetworkStatusLimited // Can reach some endpoints but not others
)

// String returns the string representation of NetworkStatus
func (ns NetworkStatus) String() string {
	switch ns {
	case NetworkStatusOnline:
		return "online"
	case NetworkStatusOffline:
		return "offline"
	case NetworkStatusLimited:
		return "limited"
	default:
		return "unknown"
	}
}

// NetworkMonitor monitors network connectivity and provides status updates
type NetworkMonitor struct {
	status          NetworkStatus
	lastCheck       time.Time
	checkInterval   time.Duration
	timeout         time.Duration
	maxTimeout      time.Duration
	testEndpoints   []string
	logger          *Logger
	mu              sync.RWMutex
	statusCallbacks []func(NetworkStatus)
	httpClient      *http.Client
	isMonitoring    bool
	stopChan        chan struct{}
}

// GetStatus returns the current network status
func (nm *NetworkMonitor) GetStatus() NetworkStatus {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.status
}

// GetLastCheck returns the time of the last connectivity check
func (nm *NetworkMonitor) GetLastCheck() time.Time {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.lastCheck
}

// IsOnline returns true if the network is considered online
func (nm *NetworkMonitor) IsOnline() bool {
	return nm.GetStatus() == NetworkStatusOnline
}

// IsOffline returns true if the network is considered offline
func (nm *NetworkMonitor) IsOffline() bool {
	status := nm.GetStatus()
	return status == NetworkStatusOffline || status == NetworkStatusUnknown
}

// SetTimeout updates the network check timeout (up to MaxTimeout)
func (nm *NetworkMonitor) SetTimeout(timeout time.Duration) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if timeout > nm.maxTimeout {
		timeout = nm.maxTimeout
	}
	if timeout < time.Second {
		timeout = time.Second
	}

	nm.timeout = timeout
	nm.httpClient.Timeout = timeout
	nm.logger.Info("Network timeout updated to %v", timeout)
}

// GetTimeout returns the current network timeout
func (nm *NetworkMonitor) GetTimeout() time.Duration {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.timeout
}

// AddStatusCallback registers a callback for status changes
func (nm *NetworkMonitor) AddStatusCallback(callback func(NetworkStatus)) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.statusCallbacks = append(nm.statusCallbacks, callback)
}

// CheckConnectivity performs an immediate connectivity check
func (nm *NetworkMonitor) CheckConnectivity(ctx context.Context) NetworkStatus {
	nm.logger.Info("Checking network connectivity...")

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, nm.timeout)
	defer cancel()

	onlineCount := 0
	totalChecks := len(nm.testEndpoints)

	for _, endpoint := range nm.testEndpoints {
		if nm.checkEndpoint(checkCtx, endpoint) {
			onlineCount++
		}
	}

	var newStatus NetworkStatus
	switch onlineCount {
	case 0:
		newStatus = NetworkStatusOffline
	case totalChecks:
		newStatus = NetworkStatusOnline
	default:
		newStatus = NetworkStatusLimited
	}

	nm.updateStatus(newStatus)
	nm.logger.Info("Connectivity check complete: %s (%d/%d endpoints reachable)",
		newStatus.String(), onlineCount, totalChecks)

	return newStatus
}

// checkEndpoint tests connectivity to a specific endpoint
func (nm *NetworkMonitor) checkEndpoint(ctx context.Context, endpoint string) bool {
	req, err := http.NewRequestWithContext(ctx, "HEAD", endpoint, nil)
	if err != nil {
		nm.logger.Error("Failed to create request for %s: %v", endpoint, err)
		return false
	}

	// Set minimal headers to avoid authentication issues
	req.Header.Set("User-Agent", "Go-AIProvider-NetworkCheck/1.0")

	resp, err := nm.httpClient.Do(req)
	if err != nil {
		nm.logger.Warn("Endpoint %s unreachable: %v", endpoint, err)
		return false
	}
	defer resp.Body.Close()

	// Consider any response (even errors) as connectivity
	// We're just checking if we can reach the endpoint
	nm.logger.Info("Endpoint %s reachable (status: %d)", endpoint, resp.StatusCode)
	return true
}

// updateStatus updates the network status and notifies callbacks
func (nm *NetworkMonitor) updateStatus(newStatus NetworkStatus) {
	nm.mu.Lock()
	oldStatus := nm.status
	nm.status = newStatus
	nm.lastCheck = time.Now()
	callbacks := make([]func(NetworkStatus), len(nm.statusCallbacks))
	copy(callbacks, nm.statusCallbacks)
	nm.mu.Unlock()

	// Notify callbacks if status changed
	if oldStatus != newStatus {
		nm.logger.Info("Network status changed: %s -> %s", oldStatus.String(), newStatus.String())
		for _, callback := range callbacks {
			go func(cb func(NetworkStatus)) {
				defer func() {
					if r := recover(); r != nil {
						nm.logger.Error("Status callback panicked: %v", r)
					}
				}()
				cb(newStatus)
			}(callback)
		}
	}
}

// StartMonitoring begins continuous network monitoring
func (nm *NetworkMonitor) StartMonitoring(ctx context.Context) {
	nm.mu.Lock()
	if nm.isMonitoring {
		nm.mu.Unlock()
		return
	}
	nm.isMonitoring = true
	nm.mu.Unlock()

	nm.logger.Info("Starting network monitoring (interval: %v, timeout: %v)",
		nm.checkInterval, nm.timeout)

	// Perform initial check
	nm.CheckConnectivity(ctx)

	// Start monitoring loop
	go nm.monitoringLoop(ctx)
}

// StopMonitoring stops continuous network monitoring
func (nm *NetworkMonitor) StopMonitoring() {
	nm.mu.Lock()
	if !nm.isMonitoring {
		nm.mu.Unlock()
		return
	}
	nm.isMonitoring = false
	nm.mu.Unlock()

	nm.logger.Info("Stopping network monitoring")
	close(nm.stopChan)
}

// monitoringLoop runs the continuous monitoring process
func (nm *NetworkMonitor) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(nm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			nm.logger.Info("Network monitoring stopped due to context cancellation")
			return
		case <-nm.stopChan:
			nm.logger.Info("Network monitoring stopped")
			return
		case <-ticker.C:
			nm.CheckConnectivity(ctx)
		}
	}
}

// GetNetworkInfo returns detailed network information
func (nm *NetworkMonitor) GetNetworkInfo() map[string]any {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	return map[string]any{
		"status":        nm.status.String(),
		"lastCheck":     nm.lastCheck.Format(time.RFC3339),
		"timeout":       nm.timeout.String(),
		"maxTimeout":    nm.maxTimeout.String(),
		"checkInterval": nm.checkInterval.String(),
		"isMonitoring":  nm.isMonitoring,
		"endpoints":     nm.testEndpoints,
	}
}

// TestEndpointConnectivity tests connectivity to a specific endpoint with custom timeout
func (nm *NetworkMonitor) TestEndpointConnectivity(ctx context.Context, endpoint string, timeout time.Duration) error {
	if timeout > nm.maxTimeout {
		timeout = nm.maxTimeout
	}

	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(testCtx, "HEAD", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Go-AIProvider-NetworkTest/1.0")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	nm.logger.Info("Endpoint %s test successful (status: %d)", endpoint, resp.StatusCode)
	return nil
}

// NewNetworkMonitor creates a new network monitor instance
func NewNetworkMonitor(config NetworkMonitorConfig) *NetworkMonitor {
	// Set defaults
	if config.CheckInterval == 0 {
		config.CheckInterval = 30 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.MaxTimeout == 0 {
		config.MaxTimeout = 10 * time.Second
	}
	if len(config.TestEndpoints) == 0 {
		config.TestEndpoints = []string{
			"https://api.anthropic.com/v1/messages",
			"https://api.openai.com/v1/chat/completions",
			"https://www.google.com",
		}
	}

	// Ensure timeout doesn't exceed max
	if config.Timeout > config.MaxTimeout {
		config.Timeout = config.MaxTimeout
	}

	return &NetworkMonitor{
		status:          NetworkStatusUnknown,
		checkInterval:   config.CheckInterval,
		timeout:         config.Timeout,
		maxTimeout:      config.MaxTimeout,
		testEndpoints:   config.TestEndpoints,
		logger:          NewLogger("NetworkMonitor"),
		statusCallbacks: make([]func(NetworkStatus), 0),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stopChan: make(chan struct{}),
	}
}

// Global network monitor instance
var globalNetworkMonitor *NetworkMonitor

// InitializeNetworkMonitor initializes the global network monitor
func InitializeNetworkMonitor(config NetworkMonitorConfig) {
	globalNetworkMonitor = NewNetworkMonitor(config)
}

// GetGlobalNetworkMonitor returns the global network monitor instance
func GetGlobalNetworkMonitor() *NetworkMonitor {
	return globalNetworkMonitor
}
