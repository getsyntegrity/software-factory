//go:build unix

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// lockJob blocks until it holds an exclusive flock on <root>/.<id>.lock and
// returns the function that releases it. The kernel releases the lock when
// the holding process exits or dies, so a crash never leaves it stale.
//
// The release removes the lock file before unlocking it. A waiter that then
// gets the lock on the removed file sees that the path no longer names its
// file and starts over, so two holders never lock different files.
func lockJob(root, id string) (func(), error) {
	path := filepath.Join(root, "."+id+".lock")
	for {
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			return nil, fmt.Errorf("store: lock job %q: %w", id, err)
		}
		if err := flock(f); err != nil {
			f.Close()
			return nil, fmt.Errorf("store: lock job %q: %w", id, err)
		}
		held, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("store: lock job %q: %w", id, err)
		}
		if current, err := os.Stat(path); err == nil && os.SameFile(held, current) {
			return func() {
				os.Remove(path)
				f.Close()
			}, nil
		}
		f.Close()
	}
}

func flock(f *os.File) error {
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
		if err != syscall.EINTR {
			return err
		}
	}
}
