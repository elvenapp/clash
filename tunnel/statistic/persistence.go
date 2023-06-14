//go:build foss

package statistic

import (
	"runtime"
	"unsafe"

	C "clash-foss/constant"
	"clash-foss/log"
)

const StoreVersion = 3

type statistics struct {
	version uint64

	tracked          uint64
	directUploaded   uint64
	directDownloaded uint64
	proxyUploaded    uint64
	proxyDownloaded  uint64
}

type Persistence struct {
	statistics *statistics

	fd     uintptr
	mapped []byte
}

func Initialize() {
	path := C.Path.Resolve("statistics.dat")

	fd, mapped, err := mapFile(path, int(unsafe.Sizeof(statistics{})))
	if err != nil {
		log.Warnln("Map statistics.dat failed: %w", err)

		return
	}

	s := (*statistics)(unsafe.Pointer(&mapped[0]))
	if s.version != StoreVersion {
		log.Debugln("Upgrade statistics.dat: %d -> %d", s.version, StoreVersion)

		*s = statistics{version: StoreVersion}
	}

	p := &Persistence{
		statistics: s,
		fd:         fd,
		mapped:     mapped,
	}

	runtime.SetFinalizer(p, finalizePersistence)

	DefaultManager.persistence = p
}

func finalizePersistence(p *Persistence) {
	unmapFile(p.fd, p.mapped)
}
