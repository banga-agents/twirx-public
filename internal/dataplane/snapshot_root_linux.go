//go:build linux

package dataplane

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

type linuxSnapshotRoot struct {
	directory *os.File
	device    uint64
}

func openSnapshotRoot(directory string) (snapshotRoot, error) {
	root, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	info, err := root.Stat()
	if err != nil {
		root.Close()
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() {
		root.Close()
		return nil, fmt.Errorf("%w: snapshot root is not a directory", ErrInvalid)
	}
	return &linuxSnapshotRoot{directory: root, device: uint64(stat.Dev)}, nil
}

func (r *linuxSnapshotRoot) Close() error {
	return r.directory.Close()
}

func (r *linuxSnapshotRoot) OpenRegular(name string) (*os.File, os.FileInfo, error) {
	if err := validateSnapshotOpenPath(name); err != nil {
		return nil, nil, err
	}
	segments := strings.Split(name, "/")
	parentFD, err := syscall.Dup(int(r.directory.Fd()))
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if parentFD >= 0 {
			_ = syscall.Close(parentFD)
		}
	}()

	for _, segment := range segments[:len(segments)-1] {
		nextFD, openErr := syscall.Openat(parentFD, segment, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
		if openErr != nil {
			return nil, nil, openErr
		}
		var stat syscall.Stat_t
		if statErr := syscall.Fstat(nextFD, &stat); statErr != nil || stat.Mode&syscall.S_IFMT != syscall.S_IFDIR || uint64(stat.Dev) != r.device {
			_ = syscall.Close(nextFD)
			if statErr != nil {
				return nil, nil, statErr
			}
			return nil, nil, fmt.Errorf("%w: unsafe directory component in %q", ErrInvalid, name)
		}
		_ = syscall.Close(parentFD)
		parentFD = nextFD
	}

	fileFD, err := syscall.Openat(parentFD, segments[len(segments)-1], syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, err
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(fileFD, &stat); err != nil {
		_ = syscall.Close(fileFD)
		return nil, nil, err
	}
	if stat.Mode&syscall.S_IFMT != syscall.S_IFREG || stat.Nlink != 1 || uint64(stat.Dev) != r.device {
		_ = syscall.Close(fileFD)
		return nil, nil, fmt.Errorf("%w: %q is not an admitted regular file", ErrInvalid, name)
	}
	file := os.NewFile(uintptr(fileFD), name)
	if file == nil {
		_ = syscall.Close(fileFD)
		return nil, nil, fmt.Errorf("%w: cannot wrap snapshot file %q", ErrInvalid, name)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	return file, info, nil
}
