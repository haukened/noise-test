package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/perlin-network/noise"
)

var nodeOpts []noise.NodeOption

func init() {
	var nodeAddrStr string
	var nodePort uint
	flag.StringVar(&nodeAddrStr, "address", "", "if desired, specify a system IP address to bind to. Otherwise, noise will listen on all IP addresses.")
	flag.UintVar(&nodePort, "port", 0, "port for this node to listen on. Otherwise noise will pick a random port to listen on.")
	flag.Parse()

	// then parse the IP adress
	nodeIP := net.ParseIP(nodeAddrStr)
	// if nodeIP is nil, the address was not parseable, or not specified
	if nodeIP != nil {
		nodeOpts = append(nodeOpts, noise.WithNodeBindHost(nodeIP))
	}

	// then convert the int to uint16
	if nodePort > 0 && nodePort <= 65535 {
		nodeOpts = append(nodeOpts, noise.WithNodeBindPort(uint16(nodePort)))
	} else if nodePort > 65535 {
		fmt.Fprintf(os.Stderr, "%d is not a valid port number (1-65535)\n", nodePort)
		os.Exit(1)
	}
}
