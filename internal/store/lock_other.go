//go:build !unix

package store

import "errors"

// lockJob is not supported off Unix: there is no per-job lock across
// processes, so operations that need one fail instead of running unlocked.
func lockJob(root, id string) (func(), error) {
	return nil, errors.New("store: per-job locking is not supported on this platform")
}
