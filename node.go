package main

import (
	"context"
	"time"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/kademlia"
	"go.uber.org/zap"
)

// ServerNode holds all required information and options for a distributed chat node service
type ServerNode struct {
	Host              *noise.Node
	Kademlia          *kademlia.Protocol
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
		// opts.NodeOpts = append(opts.NodeOpts, noise.WithNodeLogger(opts.Logger))
	}
	var err error
	// initialize the host
	n.Host, err = noise.NewNode(opts.NodeOpts...)
	if err != nil {
		return err
	}
	// create a new kademlia discovery protocol
	n.Kademlia = kademlia.New()
	// bind the discovery protocol to the host
	n.Host.Bind(n.Kademlia.Protocol())
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
	n.DiscoveredPeers = n.Kademlia.Discover()
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
			case <-n.stopDiscovery:
				break outer
			case _ = <-ticker.C:
				s.Discover()
				n.Log.Debugf("discovered %d peers", len(s.DiscoveredPeers))
			}
		}
		n.Log.Info("peer discovery stopped")
	}(n)
}

// StopDiscovery stops the kademlia discovery process
func (n *ServerNode) StopDiscovery() {
	n.Log.Debug("peer discovery stopping")
	n.stopDiscovery <- true
	n.DiscoveryActive = false
}
