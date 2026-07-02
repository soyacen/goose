package ws

import (
	"context"
	"errors"

	"github.com/coder/websocket"
)

// AcceptOptions returns the default websocket.AcceptOptions for upgrading
// HTTP connections to WebSocket. Override InsecureSkipVerify to disable
// CORS origin checking if needed.
func AcceptOptions() *websocket.AcceptOptions {
	return &websocket.AcceptOptions{
		InsecureSkipVerify: false,
	}
}

// IsNormalClose returns true if the error represents a normal WebSocket close.
// This includes StatusNormalClosure, StatusGoingAway, and context.Canceled.
func IsNormalClose(err error) bool {
	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure ||
		status == websocket.StatusGoingAway ||
		errors.Is(err, context.Canceled)
}
