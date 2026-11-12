package transports

import (
	"strings"
)

func listTransports() []string {
	return []string{
		"connectrpc.com",
		"database/sql",
		"net/http",
		"github.com/jiva-studio/numen/libs/core/protocol",
	}
}

func isTransport(to string) bool {
	for _, one := range listTransports() {
		if to == one || strings.HasPrefix(to, one+"/") {
			return true
		}
	}
	return false
}
