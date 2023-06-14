//go:build foss

package mixed

import (
	"net"

	"clash-foss/common/cache"
	N "clash-foss/common/net"
	"clash-foss/component/auth"
	C "clash-foss/constant"
	authStore "clash-foss/listener/auth"
	"clash-foss/listener/http"
	"clash-foss/listener/socks"
	"clash-foss/transport/socks4"
	"clash-foss/transport/socks5"
)

type Listener struct {
	listener net.Listener
	addr     string
	cache    *cache.LruCache
	closed   bool
}

// RawAddress implements C.Listener
func (l *Listener) RawAddress() string {
	return l.addr
}

// Address implements C.Listener
func (l *Listener) Address() string {
	return l.listener.Addr().String()
}

// Close implements C.Listener
func (l *Listener) Close() error {
	l.closed = true
	return l.listener.Close()
}

func New(addr string, in chan<- C.ConnContext) (*Listener, error) {
	return NewWithAuthenticator(addr, in, authStore.Authenticator())
}

func NewWithAuthenticator(addr string, in chan<- C.ConnContext, authenticator auth.Authenticator) (*Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ml := &Listener{
		listener: l,
		addr:     addr,
		cache:    cache.New(cache.WithAge(30)),
	}
	go func() {
		for {
			c, err := ml.listener.Accept()
			if err != nil {
				if ml.closed {
					break
				}
				continue
			}
			go handleConn(c, in, authenticator)
		}
	}()

	return ml, nil
}

func handleConn(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	//conn.(*net.TCPConn).SetKeepAlive(true)

	bufConn := N.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		return
	}

	switch head[0] {
	case socks4.Version:
		socks.HandleSocks4(bufConn, in, authenticator)
	case socks5.Version:
		socks.HandleSocks5(bufConn, in, authenticator)
	default:
		http.HandleConn(bufConn, in, authenticator)
	}
}
