//go:build foss

package statistic

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"

	"clash-foss/component/ring"
	C "clash-foss/constant"
)

const (
	MaxActiveTCPConnection = 512
	MaxActiveUDPConnection = 256
	MaxConnectionHistory   = 1024
)

var (
	DefaultManager = &Manager{
		connections: map[uuid.UUID]*tracking{},
		history:     ring.NewRing[*TrackerInfo](MaxConnectionHistory),
	}

	defaultStatistics = statistics{}
)

type Manager struct {
	lock sync.Mutex

	tcpConnections uint64
	udpConnections uint64
	connections    map[uuid.UUID]*tracking
	history        *ring.Ring[*TrackerInfo]

	persistence *Persistence
}

func (m *Manager) TrackConn(conn C.Conn, metadata *C.Metadata, rule C.Rule) (C.Conn, error) {
	tk, err := m.track(conn, conn.Chains(), metadata, rule)
	if err != nil {
		return nil, err
	}

	return &trackableConn{
		Conn:    conn,
		tracker: tk,
	}, nil
}

func (m *Manager) TrackPacketConn(conn C.PacketConn, metadata *C.Metadata, rule C.Rule) (C.PacketConn, error) {
	tk, err := m.track(conn, conn.Chains(), metadata, rule)
	if err != nil {
		return nil, err
	}

	return &trackablePacketConn{
		PacketConn: conn,
		tracker:    tk,
	}, nil
}

func (m *Manager) BandwidthDirect() (up, down uint64) {
	s := m.statistics()

	up = atomic.LoadUint64(&s.directUploaded)
	down = atomic.LoadUint64(&s.directDownloaded)

	return
}

func (m *Manager) BandwidthProxy() (up, down uint64) {
	s := m.statistics()

	up = atomic.LoadUint64(&s.proxyUploaded)
	down = atomic.LoadUint64(&s.proxyDownloaded)

	return
}

func (m *Manager) ConnectionsCount() (now, history uint64) {
	s := m.statistics()

	now = atomic.LoadUint64(&m.tcpConnections) + atomic.LoadUint64(&m.udpConnections)
	history = atomic.LoadUint64(&s.tracked)

	return
}

func (m *Manager) ResetBandwidth() {
	s := m.statistics()

	atomic.StoreUint64(&s.directUploaded, 0)
	atomic.StoreUint64(&s.directDownloaded, 0)
	atomic.StoreUint64(&s.proxyUploaded, 0)
	atomic.StoreUint64(&s.proxyDownloaded, 0)
}

func (m *Manager) ResetConnections() {
	s := m.statistics()

	m.lock.Lock()
	defer m.lock.Unlock()

	atomic.StoreUint64(&s.tracked, m.tcpConnections+m.udpConnections)
}

func (m *Manager) Snapshot() *Snapshot {
	m.lock.Lock()
	defer m.lock.Unlock()

	connections := make([]*TrackerInfo, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn.info)
	}

	directUploadTotal, directDownloadTotal := m.BandwidthDirect()
	proxyUploadTotal, proxyDownloadTotal := m.BandwidthDirect()

	return &Snapshot{
		UploadTotal:   directUploadTotal + proxyUploadTotal,
		DownloadTotal: directDownloadTotal + proxyDownloadTotal,
		Connections:   connections,
	}
}

func (m *Manager) CloseConnection(id uuid.UUID) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	conn, ok := m.connections[id]
	if ok {
		return true, conn.conn.Close()
	}

	return false, nil
}

func (m *Manager) HistoryFirst() int {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.history.Position()
}

func (m *Manager) HistoryLast() int {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.history.Limit()
}

func (m *Manager) DumpHistory(index int, out []*TrackerInfo) (int, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	n, _, ok := m.history.Get(index, out)
	return n, ok
}

func (m *Manager) InstallPersistence(p *Persistence) {
	m.persistence = p
}

func (m *Manager) statistics() *statistics {
	p := m.persistence
	if p != nil {
		return p.statistics
	}

	return &defaultStatistics
}

func (m *Manager) track(conn io.Closer, chain C.Chain, metadata *C.Metadata, rule C.Rule) (*tracker, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	tcp := metadata.NetWork == C.TCP

	if tcp {
		if m.tcpConnections > MaxActiveTCPConnection {
			return nil, errors.New("connections limit exceeded")
		}
	} else {
		if m.udpConnections > MaxActiveUDPConnection {
			return nil, errors.New("connections limit exceeded")
		}
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	s := m.statistics()

	var upload *uint64
	var download *uint64
	if chain.Last() == "DIRECT" {
		upload = &s.directUploaded
		download = &s.directDownloaded
	} else {
		upload = &s.proxyUploaded
		download = &s.proxyDownloaded
	}

	atomic.AddUint64(&s.tracked, 1)

	t := &tracker{
		info: &TrackerInfo{
			UUID:     id,
			Metadata: metadata,
			Start:    time.Now(),
			End:      nil,
			Chain:    chain,
		},
		upload:   upload,
		download: download,
		dispose: func() {
			m.lock.Lock()
			defer m.lock.Unlock()

			delete(m.connections, id)

			if tcp {
				m.tcpConnections--
			} else {
				m.udpConnections--
			}
		},
	}

	if rule != nil {
		t.info.Rule = rule.RuleType().String()
		t.info.RulePayload = rule.Payload()
	}

	m.history.Append([]*TrackerInfo{t.info})

	m.connections[id] = &tracking{
		conn: conn,
		info: t.info,
	}
	if tcp {
		m.tcpConnections++
	} else {
		m.udpConnections++
	}

	return t, nil
}

type Snapshot struct {
	DownloadTotal uint64         `json:"downloadTotal"`
	UploadTotal   uint64         `json:"uploadTotal"`
	Connections   []*TrackerInfo `json:"connections"`
}
