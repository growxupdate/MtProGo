package errors

import (
	"fmt"
	"time"
)

// FloodWaitError represents a Telegram flood wait.
type FloodWaitError struct {
	Duration time.Duration
}

func (e FloodWaitError) Error() string { return fmt.Sprintf("flood wait: %s", e.Duration) }
