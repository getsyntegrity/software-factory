// Package store persists jobs as one JSON file each under a root directory.
// Creating a job is idempotent per request ID and every write is atomic.
package store
