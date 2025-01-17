//go:build foss

package dialer

import "go.uber.org/atomic"

var (
	DefaultOptions     []Option
	DefaultInterface   = atomic.NewString("")
	DefaultRoutingMark = atomic.NewInt32(0)
)

type option struct {
	interfaceName string
	fallbackBind  bool
	addrReuse     bool
	routingMark   int
	protectSocket func(uintptr) error
}

type Option func(opt *option)

func WithInterface(name string) Option {
	return func(opt *option) {
		opt.interfaceName = name
	}
}

func WithFallbackBind(fallback bool) Option {
	return func(opt *option) {
		opt.fallbackBind = fallback
	}
}

func WithAddrReuse(reuse bool) Option {
	return func(opt *option) {
		opt.addrReuse = reuse
	}
}

func WithRoutingMark(mark int) Option {
	return func(opt *option) {
		opt.routingMark = mark
	}
}

func WithProtectSocket(protectSocket func(fd uintptr) error) Option {
	return func(opt *option) {
		opt.protectSocket = protectSocket
	}
}
