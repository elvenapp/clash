//go:build foss

package statistic

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func mapFile(path string, size int) (uintptr, []byte, error) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CREAT, 0o600)
	if err != nil {
		return 0, nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		if err != nil {
			_ = unix.Close(fd)
		}
	}()

	err = unix.FcntlFlock(uintptr(fd), unix.F_SETLK, &unix.Flock_t{
		Type:   unix.F_WRLCK,
		Whence: unix.SEEK_SET,
		Start:  0,
		Len:    int64(size),
	})
	if err != nil {
		return 0, nil, fmt.Errorf("lock %s: %w", path, err)
	}

	err = unix.Ftruncate(fd, int64(size))
	if err != nil {
		return 0, nil, fmt.Errorf("truncate %s: %w", path, err)
	}

	mapped, err := unix.Mmap(fd, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return 0, nil, fmt.Errorf("map %s: %w", path, err)
	}

	return uintptr(fd), mapped, nil
}

func unmapFile(fd uintptr, mapped []byte) {
	_ = unix.Munmap(mapped)
	_ = unix.Close(int(fd))
}
