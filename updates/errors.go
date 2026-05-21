package updates

import "errors"

// ErrNoReplySender means the message is not attached to a live client.
var ErrNoReplySender = errors.New("message has no reply sender")
