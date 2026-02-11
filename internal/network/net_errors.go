package network

import (
	"errors"
	"net"
	"strings"
)

// isExpectedNetCloseError reports network-close errors caused by intentional shutdown.
func isExpectedNetCloseError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, net.ErrClosed) {
		return true
	}

	// Keep compatibility with platforms/runtime versions that still return this text.
	return strings.Contains(err.Error(), "use of closed network connection")
}
