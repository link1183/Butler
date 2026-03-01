package obs

import (
	"errors"
	"net"
	"syscall"
	"testing"

	"github.com/gorilla/websocket"
)

func TestManagerShouldReconnect(t *testing.T) {
	manager := NewManager("localhost:4455", "", nil)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "websocket close",
			err:  &websocket.CloseError{Code: websocket.CloseAbnormalClosure},
			want: true,
		},
		{
			name: "network op error",
			err:  &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
			want: true,
		},
		{
			name: "client already disconnected",
			err:  errors.New("request SetCurrentProgramScene: client already disconnected"),
			want: true,
		},
		{
			name: "closed network connection",
			err:  errors.New("write tcp 127.0.0.1: use of closed network connection"),
			want: true,
		},
		{
			name: "obs request validation failure",
			err:  errors.New("request SetCurrentProgramScene: resource not found (600)"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.shouldReconnect(tt.err)
			if got != tt.want {
				t.Fatalf("shouldReconnect(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
