// Package host provides gRPC client functionality for communicating with the Cline extension.
package host

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultProtoBusPort is the default port for the core extension gRPC service
	DefaultProtoBusPort = 26040
	// DefaultHostBridgePort is the default port for the host bridge service
	DefaultHostBridgePort = 35453
)

// EndpointConfig holds configuration for connecting to the Cline core extension
type EndpointConfig struct {
	// Address is the gRPC server address (host:port)
	Address string
	// HostBridgeAddress is the host bridge address (host:port)
	HostBridgeAddress string
}

// EndpointResolver resolves the gRPC endpoint for the Cline core extension
type EndpointResolver struct {
	dataDir string
}

// NewEndpointResolver creates a new endpoint resolver
func NewEndpointResolver(dataDir string) *EndpointResolver {
	if dataDir == "" {
		dataDir = getDefaultDataDir()
	}
	return &EndpointResolver{dataDir: dataDir}
}

// Resolve determines the gRPC endpoint address using the following priority:
// 1. Environment variable PROTOBUS_ADDRESS
// 2. Lock file in ~/.cline/data/locks.db (instance registration)
// 3. Default localhost:26040
func (r *EndpointResolver) Resolve() (*EndpointConfig, error) {
	// Priority 1: Environment variable
	if addr := os.Getenv("PROTOBUS_ADDRESS"); addr != "" {
		hostBridgeAddr := os.Getenv("HOST_BRIDGE_ADDRESS")
		if hostBridgeAddr == "" {
			hostBridgeAddr = fmt.Sprintf("localhost:%d", DefaultHostBridgePort)
		}
		return &EndpointConfig{
			Address:           addr,
			HostBridgeAddress: hostBridgeAddr,
		}, nil
	}

	// Priority 2: Try to read from lock file/registry
	if addr, err := r.resolveFromLockFile(); err == nil && addr != "" {
		return &EndpointConfig{
			Address:           addr,
			HostBridgeAddress: fmt.Sprintf("localhost:%d", DefaultHostBridgePort),
		}, nil
	}

	// Priority 3: Default
	return &EndpointConfig{
		Address:           fmt.Sprintf("localhost:%d", DefaultProtoBusPort),
		HostBridgeAddress: fmt.Sprintf("localhost:%d", DefaultHostBridgePort),
	}, nil
}

// resolveFromLockFile attempts to read the active instance address from the lock database
func (r *EndpointResolver) resolveFromLockFile() (string, error) {
	// This is a simplified implementation. In production, you'd read from SQLite
	// For now, we just return empty to fall back to default
	return "", fmt.Errorf("lock file resolution not implemented")
}

// getDefaultDataDir returns the default Cline data directory
func getDefaultDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".cline", "data")
}

// ParseAddress parses a host:port address string
func ParseAddress(addr string) (host string, port int, err error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format: %s", addr)
	}
	
	host = parts[0]
	if host == "" {
		host = "localhost"
	}
	
	_, err = fmt.Sscanf(parts[1], "%d", &port)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", parts[1])
	}
	
	return host, port, nil
}