package benchmark

import (
	"context"
	"time"

	net "github.com/libp2p/go-libp2p-kad-dht/net"
	pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// MsgNotifier represents the communication channel or the debugging channel
// for the Hoarder to identify the messages sent over the DHT messenger
type MsgNotifier struct {
	msgChan chan *MsgNotification
}

func NewMsgNotifier() *MsgNotifier {
	return &MsgNotifier{
		msgChan: make(chan *MsgNotification),
	}
}

func (n *MsgNotifier) GetNotifierChan() chan *MsgNotification {
	return n.msgChan
}

func (n *MsgNotifier) Notify(msgStatus *MsgNotification) {
	n.msgChan <- msgStatus
}

func (n *MsgNotifier) Close() {
	close(n.msgChan)
}

// MsgNotification is the basic notification struct received and send by the dht messenger
type MsgNotification struct {
	RemotePeer    peer.ID
	QueryTime     time.Time
	QueryDuration time.Duration
	Msg           pb.Message
	Resp          pb.Message
	Error         error
}

// MessageSender handles sending wire protocol messages to a given peer
type MessageSender struct {
	m             pb.MessageSender
	blacklistedUA string
	msgNot        *MsgNotifier
}

func NewCustomMessageSender(blacklistedUA string, withMsgNot bool) *MessageSender {
	msgSender := &MessageSender{
		blacklistedUA: blacklistedUA,
	}
	// only generate a notifier if requested
	if withMsgNot {
		msgSender.msgNot = NewMsgNotifier()
	}
	return msgSender
}
func (ms *MessageSender) Init(h host.Host, protocols []protocol.ID) pb.MessageSender {
	msgSender := net.NewMessageSenderImpl(h, protocols, ms.blacklistedUA)
	ms.m = msgSender
	return ms
}

func (ms *MessageSender) GetMsgNotifier() *MsgNotifier {
	return ms.msgNot
}

func (ms *MessageSender) SendRequest(ctx context.Context, p peer.ID, pmes *pb.Message) (*pb.Message, error) {
	return ms.m.SendRequest(ctx, p, pmes)
}

// SendMessage is a custom wrapper on top of the pb.MessageSender that sends a given msg to a peer and
// notifies throught the given notification channel of the sent msg status
func (ms *MessageSender) SendMessage(ctx context.Context, p peer.ID, pmes *pb.Message) error {
	startT := time.Now()
	err := ms.m.SendMessage(ctx, p, pmes)
	t := time.Since(startT)

	// only notify if the notifier was enabled
	if ms.msgNot != nil {
		not := &MsgNotification{
			RemotePeer:    p,
			QueryTime:     startT,
			QueryDuration: t,
			Msg:           *pmes,
			Error:         err,
		}
		ms.msgNot.Notify(not)
	}
	return err
}
