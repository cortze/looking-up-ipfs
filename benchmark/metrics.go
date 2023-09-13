package benchmark

import (
	"github.com/ipfs/go-cid"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

type CidMetrics struct {
	job      int
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

func NewCidMetrics(job, id int, c cid.Cid, provider peer.ID) *CidMetrics {
	return &CidMetrics{
		job:      job,
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
	m.pingResult = r
	m.pingLookupMetrics = lkm
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

func (j *LookupJob) AddProvideTimes(startTime, finishTime time.Time) {
	j.provideStartTime = startTime
	j.provideFinishTime = finishTime
	j.provideProcDuration = finishTime.Sub(startTime)
}

func (j *LookupJob) AddPingTimes(startTime, finishTime time.Time) {
	j.pingStartTime = startTime
	j.pingFinishTime = finishTime
	j.pingDuration = finishTime.Sub(startTime)
}

func generateProvideJobSummary(j *LookupJob) (columns []string, rows [][]interface{}) {
	columns = []string{
		"job",
		"cids",
		"start_time",
		"finish_time",
		"duration_ns",
	}
	row := make([]interface{}, 0)
	row = append(row, j.id)
	row = append(row, len(j.Cids))
	row = append(row, j.provideStartTime)
	row = append(row, j.provideFinishTime)
	row = append(row, j.provideProcDuration.Nanoseconds())
	rows = append(rows, row)
	return columns, rows
}

func generateRetrievalJobSummary(j *LookupJob) (columns []string, rows [][]interface{}) {
	columns = []string{
		"job",
		"cids",
		"start_time",
		"finish_time",
		"duration_ns",
	}
	row := make([]interface{}, 0)
	row = append(row, j.id)
	row = append(row, len(j.Cids))
	row = append(row, j.pingStartTime)
	row = append(row, j.pingFinishTime)
	row = append(row, j.pingDuration.Nanoseconds())
	rows = append(rows, row)
	return columns, rows
}

func generateIdividualProvidesSummary(c *LookupJob) (columns []string, rows [][]interface{}) {
	columns = []string{
		"job",
		"id",
		"cid",
		"provider",
		"provide_time",
		"provide_duration_ns",
		"hops_to_closest",
		"total_hops",
	}
	for _, c := range c.Cids {
		row := make([]interface{}, 0)
		row = append(row, c.job)
		row = append(row, c.id)
		row = append(row, c.cid.String())
		row = append(row, c.provider.String())
		row = append(row, c.provTime)
		row = append(row, c.provProcDuration.Nanoseconds())
		row = append(row, c.provLookupMetrics.GetMinHopsForPeerSet(c.provLookupMetrics.GetClosestPeers()))
		row = append(row, c.provLookupMetrics.GetTotalHops())
		rows = append(rows, row)
	}
	return columns, rows
}

func generateIdividualRetrievalSummary(c *LookupJob) (columns []string, rows [][]interface{}) {
	columns = []string{
		"job",
		"id",
		"cid",
		"provider",
		"retrieval_time",
		"retrieval_duration_ns",
		"retriebable",
		"hops_to_closest",
		"total_hops",
	}
	for _, c := range c.Cids {
		row := make([]interface{}, 0)
		row = append(row, c.job)
		row = append(row, c.id)
		row = append(row, c.cid.String())
		row = append(row, c.provider.String())
		row = append(row, c.pingTime)
		row = append(row, c.pingProcDuration.Nanoseconds())
		row = append(row, c.pingResult)
		row = append(row, 0) // c.pingLookupMetrics.GetMinHopsForPeerSet(c.pingLookupMetrics.GetClosestPeers()))
		row = append(row, 0) // c.pingLookupMetrics.GetTotalHops())
		rows = append(rows, row)
	}
	return columns, rows
}
