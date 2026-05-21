package transport

import "context"

// Transport is the common network transport interface used by future MTProto runtime code.
type Transport interface {
	Connect(ctx context.Context, address string) error
	Send(ctx context.Context, payload []byte) error
	Recv(ctx context.Context) ([]byte, error)
	Close() error
}
