package domain

import "errors"

// ErrNotFound is the generic "resource does not exist" sentinel. Repositories
// return it when a mutation (update/delete/soft-delete) matches no row, and
// handlers translate it into an HTTP 404 so a missing resource is never
// reported as a 200 or a 500.
var ErrNotFound = errors.New("resource not found")

// ErrAlreadyExists signals a unique-constraint violation (e.g. a duplicate
// name/code). Handlers translate it into an HTTP 409.
var ErrAlreadyExists = errors.New("resource already exists")

// ErrInvalidAmount signals a value the user can fix (non-positive, non-finite,
// or so large the computed IDR base would overflow its numeric column). Mapped
// to a 400 instead of a 500.
var ErrInvalidAmount = errors.New("invalid amount")
