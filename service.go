package looking_up_ipfs

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"time"
)

type DHTLookupService struct {
	// control
	ctx context.Context

	// services
	dhtcli *DHTHost

	// study
	cidNumber int

	// exporter
}

func NewLookupService(ctx context.Context, cidNumber int) (*DHTLookupService, error) {
	cli, err := NewDHTHost(ctx, DefaultDHTcliOptions)
	if err != nil {
		return nil, errors.Wrap(err, "creating dht cli")
	}
	return &DHTLookupService{
		ctx:       ctx,
		dhtcli:    cli,
		cidNumber: cidNumber,
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

	// generate cids
	lookupJob := NewLookupJob(0) // hardcode the job to id=0
	log.Info("generating cids")
	for i := 0; i < s.cidNumber; i++ {
		contentId, err := s.dhtcli.ProvideCid()
	}

	// provide cids
	log.Info("providing cids")
	var errWg errgroup.Group
	startTime := time.Now()
	for cidStr, cidMetrics := range lookupJob.Cids {
		log.Debugf("Publishing %s to IPFS' DHT network", cidStr)
		errWg.Go(func() error {
			return s.provideSingleCid(cidMetrics)
		})
	}
	// check if any of the provides errored
	err = errWg.Wait()
	if err != nil {
		return err
	}
	finishTime := time.Now()
	provDuration := finishTime.Sub(startTime)

	// retrieve cids

	// get metrics, and put them somewhere

	// done
	return nil
}

// provideSingleCid makes sure to use the dhthost to publish a new CID keeping the relevant info in the CidMetrics
func (s *DHTLookupService) provideSingleCid(c *CidMetrics) error {

	return nil
}

// pingSingleCid uses the dhthost to pìng the given CID and keeping the relevant info in the CidMetrics
func (s *DHTLookupService) pingSingleCid(c *CidMetrics) error {

	return nil
}

// Close finishes the routines making sure to keep the relevant data
func (s *DHTLookupService) Close() {

}
