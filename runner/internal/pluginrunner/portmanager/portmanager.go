package portmanager

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	minPort     = 40000
	maxPort     = 50000
	maxAttempts = 10
)

type PortManager struct {
	mu           sync.Mutex
	usedPorts    map[int]struct{}
	lastAssigned int
}

func New() *PortManager {
	return &PortManager{
		usedPorts:    make(map[int]struct{}),
		lastAssigned: minPort - 1,
	}
}

func (pm *PortManager) GetPort() (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Try next port in sequence
		pm.lastAssigned++
		if pm.lastAssigned > maxPort {
			pm.lastAssigned = minPort
		}

		// Skip if port is already in use
		if _, exists := pm.usedPorts[pm.lastAssigned]; exists {
			continue
		}

		// Try to bind to the port
		listener, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(pm.lastAssigned)))
		if err != nil {
			continue
		}
		listener.Close()

		// Mark port as used
		pm.usedPorts[pm.lastAssigned] = struct{}{}
		return pm.lastAssigned, nil
	}

	return 0, fmt.Errorf("failed to find available port after %d attempts", maxAttempts)
}

func (pm *PortManager) ReleasePort(port int) error {
	if port < minPort || port > maxPort {
		return errors.Errorf("port %d is outside valid range (%d-%d)", port, minPort, maxPort)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.usedPorts, port)
	return nil
}

func (pm *PortManager) WaitForPortAvailable(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("", strconv.Itoa(port)), time.Second)
		if err != nil {
			// Port is not in use
			return nil
		}
		if conn != nil {
			conn.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("port %d still in use after %v", port, timeout)
}
