//go:build foss

package statistic

import (
	"io"
	"net"
	"sync/atomic"
	"time"

	C "clash-foss/constant"

	"github.com/gofrs/uuid"
)

type TrackerInfo struct {
	UUID          uuid.UUID   `json:"id"`
	Metadata      *C.Metadata `json:"metadata"`
	UploadTotal   uint64      `json:"upload"`
	DownloadTotal uint64      `json:"download"`
	Start         time.Time   `json:"start"`
	End           *time.Time  `json:"end"`
	Chain         C.Chain     `json:"chains"`
	Rule          string      `json:"rule"`
	RulePayload   string      `json:"rulePayload"`
}

type tracking struct {
	conn io.Closer
	info *TrackerInfo
}

type tracker struct {
	info     *TrackerInfo
	upload   *uint64
	download *uint64
	dispose  func()
}

type trackableConn struct {
	C.Conn
	tracker *tracker
}

func (t *tracker) pushUploaded(n uint64) {
	atomic.AddUint64(t.upload, n)

	t.info.UploadTotal += n
}

func (t *tracker) pushDownloaded(n uint64) {
	atomic.AddUint64(t.download, n)

	t.info.DownloadTotal += n
}

func (t *tracker) markClosed() {
	end := time.Now()

	t.info.End = &end
}

func (c *trackableConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	c.tracker.pushDownloaded(uint64(n))
	return n, err
}

func (c *trackableConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.tracker.pushUploaded(uint64(n))
	return n, err
}

func (c *trackableConn) Close() error {
	c.tracker.markClosed()
	c.tracker.dispose()

	return c.Conn.Close()
}

type trackablePacketConn struct {
	C.PacketConn
	tracker *tracker
}

func (c *trackablePacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.PacketConn.ReadFrom(b)
	c.tracker.pushDownloaded(uint64(n))
	return n, addr, err
}

func (c *trackablePacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	n, err := c.PacketConn.WriteTo(b, addr)
	c.tracker.pushUploaded(uint64(n))
	return n, err
}

func (c *trackablePacketConn) Close() error {
	c.tracker.markClosed()
	c.tracker.dispose()
	return c.PacketConn.Close()
}
