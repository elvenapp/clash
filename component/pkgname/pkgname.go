//go:build foss

package pkgname

import (
	"net/netip"
)

var PackageNameFinder func(network string, from netip.AddrPort, to netip.AddrPort) (string, error)
