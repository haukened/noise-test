package main

import (
	"context"
	"fmt"
	"time"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/gossip"
	"github.com/perlin-network/noise/kademlia"
	"go.uber.org/zap"
)

// ServerNode holds all required information and options for a distributed chat node service
type ServerNode struct {
	Host              *noise.Node
	Overlay           *kademlia.Protocol
	Hub               *gossip.Protocol
	DiscoveryPeer     string
	DiscoveredPeers   []noise.ID
	DiscoveryActive   bool
	DiscoveryInterval int
	stopDiscovery     chan bool
	Log               *zap.SugaredLogger
}

// ServerArgs holds all configuration options for the server
type ServerArgs struct {
	NodeOpts          []noise.NodeOption
	PeerAddress       string
	Logger            *zap.Logger
	DiscoveryInterval int
}

// Init creates a new node with the specified options, with a bound discovery protocol
func (n *ServerNode) Init(opts ServerArgs) error {
	n.stopDiscovery = make(chan bool)
	n.DiscoveryPeer = opts.PeerAddress
	n.DiscoveryInterval = opts.DiscoveryInterval
	if opts.Logger != nil {
		n.Log = opts.Logger.Sugar()
		// enabling WithNodeLogger() causes the node to echo private keys to the logs
		// DANGER DANGER DANGER
		opts.NodeOpts = append(opts.NodeOpts, noise.WithNodeLogger(opts.Logger))
	}
	var err error
	// initialize the host
	n.Host, err = noise.NewNode(opts.NodeOpts...)
	if err != nil {
		return err
	}
	// create a new kademlia discovery protocol
	n.Overlay = kademlia.New()
	// create a new gossip protocol
	n.Hub = gossip.New(n.Overlay,
		gossip.WithEvents(
			gossip.Events{
				OnGossipReceived: func(sender noise.ID, data []byte) error {
					fmt.Printf("Got a gossip from %s\n", sender.ID.String())
					return nil
				},
			},
		))
	// bind the protocols to the host
	n.Host.Bind(
		n.Overlay.Protocol(),
		n.Hub.Protocol(),
	)
	return nil
}

// Listen starts the server listening
func (n *ServerNode) Listen() error {
	return n.Host.Listen()
}

// Discover runs kademlia discovery once based on a specificed peer
func (n *ServerNode) Discover() {
	if n.DiscoveryPeer != "" {
		n.Host.Ping(context.TODO(), n.DiscoveryPeer)
	}
	n.DiscoveredPeers = n.Overlay.Discover()
}

// StartDiscovery runs the kademlia discovery process, every n number of seconds
func (n *ServerNode) StartDiscovery() {
	if n.DiscoveryActive {
		n.Log.Debug("peer discovery attempeted to start, but was already running")
		return
	}
	n.Log.Info("peer discovery is starting")
	ticker := time.NewTicker(time.Duration(n.DiscoveryInterval) * time.Second)
	go func(s *ServerNode) {
	outer:
		for {
			select {
			case <-s.stopDiscovery:
				break outer
			case _ = <-ticker.C:
				s.Discover()
				s.Log.Debugf("discovered %d peers", len(s.DiscoveredPeers))
			}
		}
		s.Log.Info("peer discovery stopped")
	}(n)
}

// StopDiscovery stops the kademlia discovery process
func (n *ServerNode) StopDiscovery() {
	n.Log.Debug("peer discovery stopping")
	n.stopDiscovery <- true
	n.DiscoveryActive = false
}

// SendGossip sends a gossip message to the entire overlay network
func (n *ServerNode) SendGossip(msg string) {
	n.Hub.Push(context.TODO(), []byte(msg))
}

// HandleGossip handles incoming gossip messages from the network
func HandleGossip(sender noise.ID, data []byte) error {
	fmt.Printf("got a gossip from %s\n", sender.ID.String())
	return nil
}

// SendMessage sends a message to a specific node
