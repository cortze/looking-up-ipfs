package benchmark

import (
	"context"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"time"
)

const (
	provideTimeout   = 80 * time.Second
	retreivalTimeout = 80 * time.Second
)

type DHTLookupService struct {
	// control
	ctx context.Context

	// services
	dhtcli *DHTHost

	// study
	jobNumber int
	cidNumber int
	jobs      []*LookupJob

	// export variables
	exportPath string
}

func NewLookupService(ctx context.Context, jobNumber, cidNumber int, exportPath string) (*DHTLookupService, error) {
	cli, err := NewDHTHost(ctx, DefaultDHTcliOptions)
	if err != nil {
		return nil, errors.Wrap(err, "creating dht cli")
	}
	return &DHTLookupService{
		ctx:        ctx,
		dhtcli:     cli,
		jobNumber:  jobNumber,
		cidNumber:  cidNumber,
		jobs:       make([]*LookupJob, 0),
		exportPath: exportPath,
	}, nil
}

func (s *DHTLookupService) Run() error {
	// set the logger
	log := logrus.WithFields(logrus.Fields{
		"host-id": s.dhtcli.ID().String()})
	log.WithField("cid-number", s.cidNumber).Info("running the Study")

	// bootstrap dht cli
	err := s.dhtcli.Init()
	if err != nil {
		return errors.Wrap(err, "initialising dht-cli")
	}
	cidProvider := s.dhtcli.host.ID()

	for j := 0; j < s.jobNumber; j++ {
		// generate cids
		lookupJob := NewLookupJob(j)
		s.jobs = append(s.jobs, lookupJob)
		log.Info("generating cids")
		for i := 0; i < s.cidNumber; i++ {
			contentId, err := s.dhtcli.GenRandomCID()
			if err != nil {
				return errors.Wrap(err, "generating random CID")
			}
			cidMetrics := NewCidMetrics(j, i, contentId, cidProvider)
			exists := lookupJob.AddCidMetrics(cidMetrics)
			if exists {
				log.Warnf("cid %s was already existed", contentId.String())
			}
		}

		// provide cids
		log.Info("providing cids")
		var errWg errgroup.Group
		startTime := time.Now()
		provide := func(m *CidMetrics) {
			errWg.Go(func() error {
				return s.provideSingleCid(m)
			})
		}
		for _, cidMetrics := range lookupJob.Cids {
			provide(cidMetrics)
		}
		err = errWg.Wait()
		if err != nil {
			return err
		}
		finishTime := time.Now()
		lookupJob.AddProvideTimes(startTime, finishTime)

		// retrieve cids
		log.Info("pinging cids")
		var pingErrWg errgroup.Group
		startTime = time.Now()
		ping := func(m *CidMetrics) {
			pingErrWg.Go(func() error {
				return s.pingSingleCid(m)
			})
		}
		for _, cidMetrics := range lookupJob.Cids {
			ping(cidMetrics)
		}
		err = pingErrWg.Wait()
		if err != nil {
			return err
		}
		finishTime = time.Now()
		lookupJob.AddPingTimes(startTime, finishTime)
	}
	// get metrics, and put them somewhere
	return s.ExportMetrics(s.exportPath)
}

// provideSingleCid makes sure to use the dhthost to publish a new CID keeping the relevant info in the CidMetrics
func (s *DHTLookupService) provideSingleCid(c *CidMetrics) error {
	log := logrus.WithFields(logrus.Fields{
		"cid": c.cid.String(),
	})
	log.Info("providing cid to the public DHT")
	ctx, cancel := context.WithTimeout(s.ctx, provideTimeout)
	defer cancel()
	startTime := time.Now()
	duration, provMetrics, err := s.dhtcli.ProvideCid(ctx, c.cid)
	if err != nil {
		return err
	}
	c.AddProvide(startTime, duration, provMetrics)
	log.WithFields(logrus.Fields{
		"duration": duration,
	}).Info("cid's PR provided to the public DHT")
	return nil
}

// pingSingleCid uses the dhthost to pìng the given CID and keeping the relevant info in the CidMetrics
func (s *DHTLookupService) pingSingleCid(c *CidMetrics) error {
	log := logrus.WithFields(logrus.Fields{
		"cid": c.cid.String(),
	})
	log.Info("retrieving cid from the public DHT")
	ctx, cancel := context.WithTimeout(s.ctx, retreivalTimeout)
	defer cancel()
	startTime := time.Now()
	duration, providers, err := s.dhtcli.FindXXProvidersOfCID(ctx, c.cid, 1)
	if err != nil {
		return err
	}
	retrievable := false
	for _, prov := range providers {
		if c.ValidProvider(prov.ID) {
			retrievable = true
			break
		}
	}
	c.AddPing(startTime, duration, retrievable, kaddht.NewLookupMetrics())
	log.WithFields(logrus.Fields{
		"retrievable": retrievable,
		"duration":    duration,
	}).Info("cid's retrieval done from the public DHT")
	return nil
}

// Close finishes the routines making sure to keep the relevant data
func (s *DHTLookupService) Close() {
	s.dhtcli.Close()
}

func (s *DHTLookupService) ExportMetrics(outputPath string) error {
	// summaries
	columns, rows := s.metricsAggregator(generateProvideJobSummary)
	err := s.exportSingleMetrics(outputPath, "provide_jobs_summary", columns, rows)
	if err != nil {
		return err
	}
	columns, rows = s.metricsAggregator(generateRetrievalJobSummary)
	err = s.exportSingleMetrics(outputPath, "retrieval_jobs_summary", columns, rows)
	if err != nil {
		return err
	}

	// individual metrics
	columns, rows = s.metricsAggregator(generateIdividualProvidesSummary)
	err = s.exportSingleMetrics(outputPath, "indv_provide_summary", columns, rows)
	if err != nil {
		return err
	}
	columns, rows = s.metricsAggregator(generateIdividualRetrievalSummary)
	err = s.exportSingleMetrics(outputPath, "indv_retrieval_summary", columns, rows)
	if err != nil {
		return err
	}
	return nil
}

func (s *DHTLookupService) exportSingleMetrics(path, appendix string, columns []string, rows [][]interface{}) error {
	exporter, err := NewCsvExporter(path+"_"+appendix+".csv", columns)
	if err != nil {
		return err
	}
	return exporter.Export(rows, StringRowComposer)
}

func (s *DHTLookupService) metricsAggregator(rowsExtractor func(j *LookupJob) ([]string, [][]interface{})) (columns []string, aggrRows [][]interface{}) {
	for i, job := range s.jobs {
		cls, rows := rowsExtractor(job)
		if i == 0 {
			columns = cls
		}
		aggrRows = append(aggrRows, rows...)
	}
	return columns, aggrRows
}
