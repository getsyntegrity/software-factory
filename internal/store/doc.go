// Package store persists jobs as one JSON file each under a root directory.
// Creating a job is idempotent per request ID and every write is atomic.
// Each job has a single writer across processes: Create and Update hold an
// exclusive per-job file lock while they read and write the job.
package store
