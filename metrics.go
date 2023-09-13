package looking_up_ipfs

import (
	"github.com/ipfs/go-cid"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

type CidMetrics struct {
	id       int
	cid      cid.Cid
	provider peer.ID

	// CID Publication/Provide
	provTime          time.Time
	provProcDuration  time.Duration
	provLookupMetrics *kaddht.LookupMetrics

	// Ping
	pingTime          time.Time
	pingProcDuration  time.Duration
	pingResult        bool // was the content retrievable?1
	pingLookupMetrics *kaddht.LookupMetrics
}

func NewCidMetrics(id int, c cid.Cid, provider peer.ID) *CidMetrics {
	return &CidMetrics{
		id:       id,
		cid:      c,
		provider: provider,
	}
}

// ValidProvider returns whether the retreived provider of the CID matches the publisher of the CID
func (m *CidMetrics) ValidProvider(r peer.ID) bool {
	return m.provider.String() == r.String()
}

func (m *CidMetrics) AddProvide(t time.Time, d time.Duration, lkm *kaddht.LookupMetrics) {
	m.provTime = t
	m.provProcDuration = d
	m.provLookupMetrics = lkm
}

func (m *CidMetrics) AddPing(t time.Time, d time.Duration, r bool, lkm *kaddht.LookupMetrics) {
	m.pingTime = t
	m.pingProcDuration = d
	m.provLookupMetrics = lkm
}

// LookupJob compiles all the data about the publication and ping of multiple CIDs concurrently
type LookupJob struct {
	id int
	// provide
	provideStartTime    time.Time
	provideProcDuration time.Duration
	provideFinishTime   time.Time
	// ping
	pingStartTime  time.Time
	pingFinishTime time.Time
	pingDuration   time.Duration
	// indv cid info
	Cids map[string]*CidMetrics
}

// NewLookupJob creates a new empty instance for aggregating the data of the study
func NewLookupJob(jobId int) *LookupJob {
	return &LookupJob{
		id:   jobId,
		Cids: make(map[string]*CidMetrics),
	}
}

func (j *LookupJob) AddCidMetrics(m *CidMetrics) (exists bool) {
	_, exists = j.Cids[m.cid.String()]
	if !exists {
		j.Cids[m.cid.String()] = m
	}
	return
}
