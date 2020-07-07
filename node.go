package main

import (
	"context"
	"time"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/kademlia"
	"github.com/sirupsen/logrus"
)

// ServerNode holds all required information and options for a distributed chat node service
type ServerNode struct {
	Host            *noise.Node
	Kademlia        *kademlia.Protocol
	DiscoveryPeer   string
	DiscoveredPeers []noise.ID
	DiscoveryActive bool
	stopDiscovery   chan bool
	Log             *logrus.Logger
}

// ServerArgs holds all configuration options for the server
type ServerArgs struct {
	nodeOpts    []noise.NodeOption
	peerAddress string
}

// Init creates a new node with the specified options, with a bound discovery protocol
func (n *ServerNode) Init(opts ServerArgs) error {
	n.stopDiscovery = make(chan bool)
	n.DiscoveryPeer = opts.peerAddress
	var err error
	// initialize the host
	n.Host, err = noise.NewNode(opts.nodeOpts...)
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
func (n *ServerNode) StartDiscovery(interval int) {
	if n.DiscoveryActive {
		n.Log.Debug("peer discovery attempeted to start, but was already running")
		return
	}
	n.Log.Info("peer discovery is starting")
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	go func(s *ServerNode) {
		for {
			select {
			case <-n.stopDiscovery:
				return
			case _ = <-ticker.C:
				s.Discover()
				n.Log.Debugf("discovered %d peers", len(s.DiscoveredPeers))
			}
		}
	}(n)
	n.Log.Info("peer discovery stopped")
}

// StopDiscovery stops the kademlia discovery process
func (n *ServerNode) StopDiscovery() {
	n.Log.Debug("peer discovery stopping")
	n.stopDiscovery <- true
	n.DiscoveryActive = false
}
