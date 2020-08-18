package e2c

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"github.com/adithyabhatkajake/libe2c/chain"
	"github.com/adithyabhatkajake/libe2c/crypto"
	"github.com/adithyabhatkajake/libe2c/util"

	"github.com/libp2p/go-libp2p"

	config "github.com/adithyabhatkajake/libe2c/config/e2c"
	"github.com/adithyabhatkajake/libe2c/net"

	msg "github.com/adithyabhatkajake/libe2c/msg/e2c"

	"github.com/libp2p/go-libp2p-core/network"
)

const (
	// ProtocolID is the ID for E2C Protocol
	ProtocolID = "e2c/e2c/0.0.1"
	// ProtocolMsgBuffer defines how many protocol messages can be buffered
	ProtocolMsgBuffer = 100
)

// Init implements the Protocol interface
func (e *E2C) Init(c *config.NodeConfig) {
	e.config = c
	e.leader = DefaultLeaderID
}

// Setup sets up the network components
func (e *E2C) Setup(n *net.Network) error {
	e.host = n.H
	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrStrings(e.config.GetClientListenAddr()),
		libp2p.Identity(e.config.GetMyKey()),
	)
	if err != nil {
		panic(err)
	}
	e.cliHost = host
	e.ctx = n.Ctx

	// Setup maps
	e.pMap = n.PeerMap
	e.streamMap = make(map[uint64]*bufio.ReadWriter)
	e.pendingCommands = make(map[crypto.Hash]*chain.Command)
	e.timerMaps = make(map[uint64]*util.Timer)

	// Setup channels
	e.msgChannel = make(chan *msg.E2CMsg, ProtocolMsgBuffer)
	e.cmdChannel = make(chan *chain.Command, ProtocolMsgBuffer)

	// Obtain a new chain
	e.bc = chain.NewChain()
	// TODO: create a new chain only if no chain is present in the data directory

	// How to react to Protocol Messages
	e.host.SetStreamHandler(ProtocolID, e.ProtoMsgHandler)

	// How to react to Client Messages
	e.cliHost.SetStreamHandler(ClientProtocolID, e.ClientMsgHandler)

	// Connect to all the other nodes talking E2C protocol
	for idx, p := range e.pMap {
		fmt.Println("Attempting to open a stream with", p, "using protocol", ProtocolID)
		retries := 300
		for i := retries; i > 0; i-- {
			s, err := e.host.NewStream(e.ctx, p.ID, ProtocolID)
			if err != nil {
				fmt.Println("Error connecting to peers:", err)
				fmt.Println("Retrying in a minute")
				<-time.After(time.Second)
				continue
			}
			e.streamMap[idx] = bufio.NewReadWriter(
				bufio.NewReader(s), bufio.NewWriter(s))
			fmt.Println("Connected to Node-#", idx)
			break
		}
	}
	fmt.Println("Setup Finished. Ready to do SMR:)")

	return nil
}

// Start implements the Protocol Interface
func (e *E2C) Start() {
	// Concurrently handle commands
	go e.cmdHandler()
	// Start E2C Protocol - Start message handler
	e.protocol()
}

// ProtoMsgHandler reacts to all protocol messages in the network
func (e *E2C) ProtoMsgHandler(s network.Stream) {
	e.errCh = make(chan error, 1)
	defer close(e.errCh)

	// Run a parallel goroutine that closes the channel if any error is detected
	go func() {
		select {
		case err, ok := <-e.errCh:
			if ok {
				fmt.Println("Type-I", err, ok)
			} else {
				fmt.Println("Type-II", err, ok)
			}
		}
		s.Reset()
	}()

	// A global buffer to collect messages
	buf := make([]byte, msg.MaxMsgSize)
	// Event Handler
	reader := bufio.NewReader(s)
	for {
		// Receive a message from anyone and process them
		len, err := reader.Read(buf)
		if err != nil {
			e.errCh <- err
			return
		}
		// Use a copy of the message and send it to off for processing
		msgBuf := make([]byte, len)
		copy(msgBuf, buf[0:len])
		// React to the message in parallel and continue
		go e.react(msgBuf)
	}
}
