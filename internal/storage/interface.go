package storage

import (
	"context"
	"errors"
)

// ErrConflict is returned when an ETag-conditional write fails.
var ErrConflict = errors.New("conflict: resource was modified")

// ErrNotFound is returned when a requested object does not exist.
var ErrNotFound = errors.New("not found")

// Store abstracts S3-compatible object storage operations.
type Store interface {
	// Get retrieves an object's data and ETag.
	Get(ctx context.Context, key string) (data []byte, etag string, err error)

	// Put writes an object. If ifMatchETag is non-empty, the write is conditional
	// (only succeeds if the current ETag matches). Returns the new ETag.
	Put(ctx context.Context, key string, data []byte, ifMatchETag string) (etag string, err error)

	// Head returns the ETag of an object without downloading its body.
	Head(ctx context.Context, key string) (etag string, err error)

	// List returns object keys matching the given prefix.
	List(ctx context.Context, prefix string) ([]string, error)

	// Delete removes an object.
	Delete(ctx context.Context, key string) error
}
