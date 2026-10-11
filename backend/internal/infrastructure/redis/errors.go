package redis

import "errors"

// ErrDisabled is returned when an operation is attempted on a disabled client
// (Redis not configured or unreachable). Callers should treat it as a signal to
// fall back to a non-cached code path.
var ErrDisabled = errors.New("redis disabled")
