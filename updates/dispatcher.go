package updates

import "context"

// MessageFilter checks whether a message should be handled.
type MessageFilter func(*Message) bool

// MessageHandler handles a message update.
type MessageHandler func(context.Context, *Message) error

type messageRoute struct {
	filter  MessageFilter
	handler MessageHandler
}

// Dispatcher routes normalized Telegram updates to handlers.
type Dispatcher struct {
	messageRoutes []messageRoute
}

// NewDispatcher creates a Dispatcher.
func NewDispatcher() *Dispatcher { return &Dispatcher{} }

// OnMessage registers a message route.
func (d *Dispatcher) OnMessage(filter MessageFilter, handler MessageHandler) {
	if handler == nil {
		return
	}
	if filter == nil {
		filter = func(*Message) bool { return true }
	}
	d.messageRoutes = append(d.messageRoutes, messageRoute{filter: filter, handler: handler})
}

// DispatchMessage dispatches one message to all matching routes.
func (d *Dispatcher) DispatchMessage(ctx context.Context, msg *Message) error {
	for _, route := range d.messageRoutes {
		if route.filter(msg) {
			if err := route.handler(ctx, msg); err != nil {
				return err
			}
		}
	}
	return nil
}
