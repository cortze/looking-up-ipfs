package looking_up_ipfs

import (
	"context"
	"fmt"
	"github.com/cortze/ipfs-cid-hoarder/pkg/models"
	"github.com/cortze/ipfs-cid-hoarder/pkg/p2p"
	mh "github.com/multiformats/go-multihash"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"

	"github.com/ipfs/go-cid"
	ma "github.com/multiformats/go-multiaddr"
)

type ProvideOption string

func GetProvOpFromConf(strOp string) ProvideOption {
	provOp := StandardDHTProvide // Default
	switch strOp {
	case "standard":
		provOp = StandardDHTProvide
	case "optimistic":
		provOp = OpDHTProvide
	}
	return provOp
}

const (
	StandardDHTProvide ProvideOption = "std-dht-provide"
	OpDHTProvide       ProvideOption = "op-provide"

	DefaultUserAgent string = "cid-hoarder"
	PingGraceTime           = 5 * time.Second
	MaxDialAttempts         = 1
	DialTimeout             = 60 * time.Second
	K                int    = 20
)

type DHTHostOptions struct {
	ID               int
	IP               string
	Port             string
	ProvOp           ProvideOption
	WithNotifier     bool
	K                int
	BlacklistingUA   string
	BlacklistedPeers map[peer.ID]struct{}
}

var DefaultDHTcliOptions = DHTHostOptions{
	ID:               0,
	IP:               "0.0.0.0",
	Port:             "9050",
	ProvOp:           StandardDHTProvide,
	WithNotifier:     false,
	K:                K,
	BlacklistingUA:   "",
	BlacklistedPeers: make(map[peer.ID]struct{}),
}

// DHT Host is the main operational instance to communicate with the IPFS DHT
type DHTHost struct {
	ctx context.Context
	m   sync.RWMutex
	// host related
	id                  int
	dht                 *kaddht.IpfsDHT
	host                host.Host
	internalMsgNotifier *p2p.MsgNotifier
	initTime            time.Time
	// dht query related
	ongoingPings map[cid.Cid]struct{}
}

func NewDHTHost(ctx context.Context, opts DHTHostOptions) (*DHTHost, error) {
	// prevent dial backoffs
	ctx = network.WithForceDirectDial(ctx, "prevent backoff")

	// Libp2p host configuration
	privKey, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate priv key for client's host")
	}
	// transport protocols
	mAddrs := make([]ma.Multiaddr, 0, 2)
	tcpAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", opts.IP, opts.Port))
	if err != nil {
		return nil, err
	}
	quicAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/%s/quic", opts.IP, opts.Port))
	if err != nil {
		return nil, err
	}
	mAddrs = append(mAddrs, tcpAddr, quicAddr)

	// kad dht options
	var dht *kaddht.IpfsDHT
	msgSender := p2p.NewCustomMessageSender(opts.BlacklistingUA, opts.WithNotifier)
	limiter := rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits)
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		return nil, fmt.Errorf("new resource manager: %w", err)
	}

	// generate the libp2p host
	h, err := libp2p.New(
		libp2p.WithDialTimeout(DialTimeout),
		libp2p.ListenAddrs(mAddrs...),
		libp2p.Identity(privKey),
		libp2p.UserAgent(DefaultUserAgent),
		libp2p.ResourceManager(rm),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.DialRanker(p2p.CustomDialRanker),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dhtOpts := make([]kaddht.Option, 0)
			dhtOpts = append(dhtOpts,
				kaddht.Mode(kaddht.ModeClient),
				kaddht.WithCustomMessageSender(msgSender.Init),
				kaddht.BucketSize(opts.K),
				kaddht.WithPeerBlacklist(opts.BlacklistedPeers)) // always blacklist (can be an empty map or a full one)
			if opts.ProvOp == OpDHTProvide {
				dhtOpts = append(dhtOpts, kaddht.EnableOptimisticProvide())
			}
			dht, err = kaddht.New(ctx, h, dhtOpts...)
			return dht, err
		}),
	)
	if err != nil {
		return nil, err
	}
	if dht == nil {
		return nil, errors.New("no IpfsDHT client was able to be created")
	}

	dhtHost := &DHTHost{
		ctx,
		sync.RWMutex{},
		opts.ID,
		dht,
		h,
		msgSender.GetMsgNotifier(),
		time.Now(),
		make(map[cid.Cid]struct{}),
	}

	err = dhtHost.Init()
	if err != nil {
		return nil, errors.Wrap(err, "unable to init host")
	}

	log.WithFields(log.Fields{
		"host-id":    opts.ID,
		"multiaddrs": h.Addrs(),
		"peer_id":    h.ID().String(),
	}).Debug("generated new host")

	return dhtHost, nil
}

// Init makes sure that all the components of the DHT host are successfully initialized
func (h *DHTHost) Init() error {
	return h.bootstrap()
}

func (h *DHTHost) bootstrap() error {
	var succCon int64
	var wg sync.WaitGroup

	hlog := log.WithField("host-id", h.id)
	for _, bnode := range kaddht.GetDefaultBootstrapPeerAddrInfos() {
		wg.Add(1)
		go func(bn peer.AddrInfo) {
			defer wg.Done()
			err := h.host.Connect(h.ctx, bn)
			if err != nil {
				hlog.Trace("unable to connect bootstrap node:", bn.String())
			} else {
				hlog.Trace("successful connection to bootstrap node:", bn.String())
				atomic.AddInt64(&succCon, 1)
			}
		}(*bnode)
	}

	wg.Wait()

	var err error
	if succCon > 0 {
		hlog.Debugf("got connected to %d bootstrap nodes", succCon)
	} else {
		err = errors.New("unable to connect any of the bootstrap nodes from KDHT")
	}
	return err
}

func (h *DHTHost) GetHostID() int {
	return h.id
}

func (h *DHTHost) GetMsgNotifier() *p2p.MsgNotifier {
	return h.internalMsgNotifier
}

// conside moving this to the Host
func (h *DHTHost) isPeerConnected(pId peer.ID) bool {
	// check if we already have a connection open to the peer
	peerList := h.host.Network().Peers()
	for _, p := range peerList {
		if p == pId {
			return true
		}
	}
	return false
}

func (h *DHTHost) GetUserAgentOfPeer(p peer.ID) (useragent string) {
	userAgentInterf, err := h.host.Peerstore().Get(p, "AgentVersion")
	if err != nil {
		useragent = p2p.NoUserAgentDefined
	} else {
		useragent = userAgentInterf.(string)
	}
	return
}

func (h *DHTHost) GetLatencyToPeer(p peer.ID) time.Duration {
	latencyItfc, err := h.host.Peerstore().Get(p, "Latency")
	latency := latencyItfc.(time.Duration)
	if err != nil {
		latency = time.Duration(0 * time.Second)
	}
	return latency
}

func (h *DHTHost) GetMAddrsOfPeer(p peer.ID) []ma.Multiaddr {
	return h.host.Peerstore().Addrs(p)
}

func (h *DHTHost) ID() peer.ID {
	return h.host.ID()
}

// dht pinger related methods
func (h *DHTHost) GetClosestPeersToCid(ctx context.Context, cid *models.CidInfo) (time.Duration, []peer.ID, *kaddht.LookupMetrics, error) {
	startT := time.Now()
	closestPeers, lookupMetrics, err := h.dht.GetClosestPeers(ctx, string(cid.CID.Hash()))
	return time.Since(startT), closestPeers, lookupMetrics, err
}

func (h *DHTHost) ProvideCid(ctx context.Context, contentID cid.Cid) (time.Duration, *kaddht.LookupMetrics, error) {

	log.WithFields(log.Fields{
		"host-id": h.id,
		"cid":     contentID.Hash().B58String(),
	}).Debug("providing cid with", h.ID().String())
	startT := time.Now()
	lookupMetrics, err := h.dht.DetailedProvide(ctx, contentID, true)
	return time.Since(startT), lookupMetrics, err
}

func (h *DHTHost) FindXXProvidersOfCID(
	ctx context.Context,
	contentID cid.Cid,
	targetProviders int) (time.Duration, []peer.AddrInfo, error) {

	log.WithFields(log.Fields{
		"host-id": h.id,
		"cid":     contentID.Hash().B58String(),
	}).Debug("looking for providers")
	startT := time.Now()
	providers, err := h.dht.LookupForXXProviders(ctx, contentID, targetProviders)
	return time.Since(startT), providers, err
}

func (h *DHTHost) FindProvidersOfCID(
	ctx context.Context,
	contentID cid.Cid) (time.Duration, []peer.AddrInfo, error) {

	log.WithFields(log.Fields{
		"host-id": h.id,
		"cid":     contentID.Hash().B58String(),
	}).Debug("looking for providers")
	startT := time.Now()
	providers, err := h.dht.LookupForProviders(ctx, contentID)
	return time.Since(startT), providers, err
}

func (h *DHTHost) Close() {
	hlog := log.WithField("host-id", h.id)
	var err error

	err = h.dht.Close()
	if err != nil {
		hlog.Error(errors.Wrap(err, "unable to close DHT client"))
	}

	err = h.host.Close()
	if err != nil {
		hlog.Error(errors.Wrap(err, "unable to close libp2p host"))
	}
	hlog.Info("successfully closed")
}

func (h *DHTHost) GenRandomCID() (cid.Cid, error) {
	content := make([]byte, 1024*1024) // 1024b * 1024 = 1KB
	rand.Read(content)

	// configure the type of CID that we want
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   mh.SHA2_256,
		MhLength: -1,
	}

	// get the CID of the content we just generated
	contID, err := pref.Sum(content)
	if err != nil {
		return cid.Cid{}, errors.Wrap(err, "composing CID")
	}
	return contID, nil
}
