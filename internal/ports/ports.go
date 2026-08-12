package ports

import (
	"net"
	"strconv"
	"sync"
)

// PortChecker defines the interface for checking port availability.
type PortChecker interface {
	IsAvailable(port int) bool
}

// RealPortChecker uses net.Listen to check port availability.
type RealPortChecker struct{}

var (
	cache     = make(map[int]bool)
	cacheMu   sync.Mutex
	cacheHits int
)

// IsAvailable checks if a TCP port is available on localhost.
// It uses net.Listen to attempt binding to the port.
func (RealPortChecker) IsAvailable(port int) bool {
	if port <= 0 || port > 65535 {
		return false
	}

	cacheMu.Lock()
	if cached, ok := cache[port]; ok {
		cacheHits++
		cacheMu.Unlock()
		return cached
	}
	cacheMu.Unlock()

	ln, err := net.Listen("tcp", "127.0.0.1:"+portString(port))
	if err != nil {
		cacheMu.Lock()
		cache[port] = false
		cacheMu.Unlock()
		return false
	}
	_ = ln.Close()

	cacheMu.Lock()
	cache[port] = true
	cacheMu.Unlock()
	return true
}

// FindAvailablePort finds the first available port starting from startPort.
// It searches up to maxAttempts ports.
func FindAvailablePort(checker PortChecker, startPort int, maxAttempts int) int {
	for i := 0; i < maxAttempts; i++ {
		port := startPort + i
		if port > 65535 {
			break
		}
		if checker.IsAvailable(port) {
			return port
		}
	}
	return startPort // fallback, should not happen in practice
}

// portString converts port int to string.
func portString(port int) string {
	return strconv.Itoa(port)
}

// MockPortChecker is a test implementation that can be configured.
type MockPortChecker struct {
	AvailablePorts map[int]bool
}

func (m MockPortChecker) IsAvailable(port int) bool {
	if m.AvailablePorts == nil {
		return true
	}
	if available, ok := m.AvailablePorts[port]; ok {
		return available
	}
	return true
}

// ResetCache clears the port availability cache. Used for testing.
func ResetCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = make(map[int]bool)
	cacheHits = 0
}
