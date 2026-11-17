// Package store persists jobs as one JSON file each under a root directory,
// <root>/<id>.json, and each job's artifacts (issue, intake, spec, build,
// decision, feedback, round checks and reviews) in the v2 layout under the
// job folder <root>/<id>/. Creating a job is idempotent per request ID and
// every write is atomic. Each job has a single writer across processes:
// Create, Update and WriteArtifact hold an exclusive per-job file lock while
// they read the job and write it or its artifacts. Sync derives the job's
// intake, gate and reviews from its artifacts and stores them under the
// job's lock, through Update.
package store
