//go:build foss

package statistic

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func mapFile(path string, size int) (uintptr, []byte, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return 0, nil, err
	}
	defer file.Close()

	fileState, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	if fileState.Size() != int64(size) {
		err = file.Truncate(int64(size))
		if err != nil {
			return 0, nil, err
		}
	}

	low, high := uint32(size), uint32(size>>32)
	fd, err := windows.CreateFileMapping(windows.Handle(file.Fd()), nil, syscall.PAGE_READWRITE, high, low, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("create mapping file %s: %w", path, err)
	}

	addr, err := windows.MapViewOfFile(fd, windows.FILE_MAP_READ|windows.FILE_MAP_WRITE, 0, 0, uintptr(size))
	if err != nil {
		return 0, nil, fmt.Errorf("map file %s: %w", path, err)
	}

	//goland:noinspection ALL
	return uintptr(fd), unsafe.Slice((*byte)(unsafe.Pointer(addr)), size), nil
}

func unmapFile(fd uintptr, mapped []byte) {
	_ = windows.UnmapViewOfFile(uintptr(unsafe.Pointer(&mapped[0])))
	_ = windows.CloseHandle(windows.Handle(fd))
}
