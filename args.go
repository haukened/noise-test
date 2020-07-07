package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/perlin-network/noise"
)

func parseArgs() (*ServerArgs, error) {
	var returnArgs ServerArgs
	var nodeAddrStr, peerAddrStr string
	var nodePort, peerPort uint
	flag.StringVar(&nodeAddrStr, "address", "", "if desired, specify a system ip address to bind to. otherwise, noise will listen on all ip addresses.")
	flag.UintVar(&nodePort, "port", 52386, "port for this node to listen on. Default is 52386.")
	flag.StringVar(&peerAddrStr, "peer-addr", "", "a destination server ip address to seed peer discovery")
	flag.UintVar(&peerPort, "peer-port", 52386, "the port the peer daemon is running on. Default is 52386")
	flag.Parse()

	// then parse the IP adress
	nodeIP := net.ParseIP(nodeAddrStr)
	// if nodeIP is nil, the address was not parseable, or not specified
	if nodeIP != nil && nodeAddrStr != "" {
		returnArgs.nodeOpts = append(returnArgs.nodeOpts, noise.WithNodeBindHost(nodeIP))
	} else if nodeIP == nil && nodeAddrStr != "" {
		// it was not parseable
		return nil, fmt.Errorf("%s is not a valid ip address", nodeAddrStr)

	}

	// then parse the port
	if nodePort > 0 && nodePort <= 65535 {
		returnArgs.nodeOpts = append(returnArgs.nodeOpts, noise.WithNodeBindPort(uint16(nodePort)))
	} else if nodePort > 65535 {
		return nil, fmt.Errorf("%d is not a valid port number (1-65535)", nodePort)
	}

	// parse peer IP address
	peerIP := net.ParseIP(peerAddrStr)
	if peerIP != nil {
		if peerPort > 0 && peerPort < 65535 {
			returnArgs.peerAddress = fmt.Sprintf("%s:%d", peerAddrStr, peerPort)
		}
	} else if peerAddrStr != "" {
		// they might have specificed a domain name, try to resolve to IP
		addr, err := net.LookupIP(peerAddrStr)
		if err != nil {
			return nil, fmt.Errorf("\"%s\" does not appear to be a valid ip address or server name", peerAddrStr)
		}
		fmt.Printf("Peer %s resolves to %s\n", peerAddrStr, addr[0].String())
		returnArgs.peerAddress = fmt.Sprintf("%s:%d", peerAddrStr, peerPort)
	}
	return &returnArgs, nil
}
