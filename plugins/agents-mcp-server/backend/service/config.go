package service

import (
	"os"
	"strings"
)

// Config configures the MCP server listener and discovery paths.
type Config struct {
	Addr          string
	Token         string
	VaultRoot     string
	DiscoveryPath string
}

// readConfig builds a configuration from values and environment variables.
func readConfig(values map[string]string) Config {
	addr := values["addr"]
	if addr == "" {
		addr = values["address"]
	}
	if addr == "" {
		if p := values["port"]; p != "" {
			addr = "127.0.0.1:" + p
		}
	}
	if addr == "" {
		if p := os.Getenv("MCP_PORT"); p != "" {
			addr = "127.0.0.1:" + p
		}
	}

	token := values["token"]
	if token == "" {
		token = os.Getenv("MCP_TOKEN")
	}

	root := values["root"]
	if root == "" {
		root = values["vault"]
	}
	if root == "" {
		root = values["vault_root"]
	}
	if root == "" {
		root = os.Getenv("NUMEN_VAULT_ROOT")
	}

	discoveryPath := values["discovery_path"]
	if discoveryPath == "" {
		discoveryPath = os.Getenv("MCP_DISCOVERY_PATH")
	}

	return Config{
		Addr:          strings.TrimSpace(addr),
		Token:         strings.TrimSpace(token),
		VaultRoot:     strings.TrimSpace(root),
		DiscoveryPath: strings.TrimSpace(discoveryPath),
	}
}
