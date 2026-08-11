//go:build !linux

package dataplane

import "fmt"

func openSnapshotRoot(string) (snapshotRoot, error) {
	return nil, fmt.Errorf("%w: secure snapshot directory verification requires Linux", ErrInvalid)
}
