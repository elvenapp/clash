//go:build foss

package http

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"clash-foss/adapter/inbound"
	N "clash-foss/common/net"
	"clash-foss/component/auth"
	C "clash-foss/constant"
)

func HandleConn(c net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	client := newClient(c.RemoteAddr(), c.LocalAddr(), in)
	defer client.CloseIdleConnections()

	conn := N.NewBufferedConn(c)

	keepAlive := true
	trusted := authenticator == nil // disable authenticate if authenticator is nil

	for keepAlive {
		request, err := ReadRequest(conn.Reader())
		if err != nil {
			break
		}

		request.RemoteAddr = conn.RemoteAddr().String()

		keepAlive = strings.TrimSpace(strings.ToLower(request.Header.Get("Proxy-Connection"))) == "keep-alive"

		var resp *http.Response

		if !trusted {
			resp = authenticate(request, authenticator)

			trusted = resp == nil
		}

		if trusted {
			if request.Method == http.MethodConnect {
				// Manual writing to support CONNECT for http 1.0 (workaround for uplay client)
				if _, err = fmt.Fprintf(conn, "HTTP/%d.%d %03d %s\r\n\r\n", request.ProtoMajor, request.ProtoMinor, http.StatusOK, "Connection established"); err != nil {
					break // close connection
				}

				in <- inbound.NewHTTPS(request, conn)

				return // hijack connection
			}

			host := request.Header.Get("Host")
			if host != "" {
				request.Host = host
			}

			request.RequestURI = ""

			if isUpgradeRequest(request) {
				handleUpgrade(conn, request, in)

				return // hijack connection
			}

			removeHopByHopHeaders(request.Header)
			removeExtraHTTPHostPort(request)

			if request.URL.Scheme == "" || request.URL.Host == "" {
				resp = responseWith(request, http.StatusBadRequest)
			} else {
				resp, err = client.Do(request)
				if err != nil {
					resp = responseWith(request, http.StatusBadGateway)
				}
			}

			removeHopByHopHeaders(resp.Header)
		}

		if keepAlive {
			resp.Header.Set("Proxy-Connection", "keep-alive")
			resp.Header.Set("Connection", "keep-alive")
			resp.Header.Set("Keep-Alive", "timeout=4")
		}

		resp.Close = !keepAlive

		err = resp.Write(conn)
		if err != nil {
			break // close connection
		}
	}

	conn.Close()
}

func authenticate(request *http.Request, authenticator auth.Authenticator) *http.Response {
	credential := parseBasicProxyAuthorization(request)
	if credential == "" {
		resp := responseWith(request, http.StatusProxyAuthRequired)
		resp.Header.Set("Proxy-Authenticate", "Basic")
		return resp
	}

	user, pass, err := decodeBasicProxyAuthorization(credential)
	if err != nil {
		return responseWith(request, http.StatusBadRequest)
	}

	if !authenticator.Verify(user, pass) {
		return responseWith(request, http.StatusForbidden)
	}

	return nil
}

func responseWith(request *http.Request, statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Proto:      request.Proto,
		ProtoMajor: request.ProtoMajor,
		ProtoMinor: request.ProtoMinor,
		Header:     http.Header{},
	}
}
