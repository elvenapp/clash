//go:build foss

package socks

import (
	"io"
	"net"

	"clash-foss/adapter/inbound"
	N "clash-foss/common/net"
	"clash-foss/component/auth"
	C "clash-foss/constant"
	authStore "clash-foss/listener/auth"
	"clash-foss/transport/socks4"
	"clash-foss/transport/socks5"
)

type Listener struct {
	listener net.Listener
	addr     string
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

	sl := &Listener{
		listener: l,
		addr:     addr,
	}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				if sl.closed {
					break
				}
				continue
			}
			go handleSocks(c, in, authenticator)
		}
	}()

	return sl, nil
}

func handleSocks(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	//conn.(*net.TCPConn).SetKeepAlive(true)
	bufConn := N.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		conn.Close()
		return
	}

	switch head[0] {
	case socks4.Version:
		HandleSocks4(bufConn, in, authenticator)
	case socks5.Version:
		HandleSocks5(bufConn, in, authenticator)
	default:
		conn.Close()
	}
}

func HandleSocks4(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	addr, _, err := socks4.ServerHandshake(conn, authenticator)
	if err != nil {
		conn.Close()
		return
	}
	in <- inbound.NewSocket(socks5.ParseAddr(addr), conn, C.SOCKS4)
}

func HandleSocks5(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	target, command, err := socks5.ServerHandshake(conn, authenticator)
	if err != nil {
		conn.Close()
		return
	}
	if command == socks5.CmdUDPAssociate {
		defer conn.Close()
		io.Copy(io.Discard, conn)
		return
	}
	in <- inbound.NewSocket(target, conn, C.SOCKS5)
}
