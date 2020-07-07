package main

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/haukened/waiter"
	"github.com/perlin-network/noise"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

// TODO: Remove designation and update this at build-time
var version = "v0.0.0"

func main() {
	if err := run(os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	app := cli.App{
		Name:                   "NoiseTest",
		HelpName:               "NoiseTest",
		Usage:                  "Brought to you by the hug-free vikings",
		UseShortOptionHandling: true,
		Writer:                 stdout,
		ErrWriter:              stderr,
		Version:                version,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "address",
				Usage: "If desired, specify a local address to bind to. Otherwise, noise will listen on all local addresses.",
			},
			&cli.UintFlag{
				Name:  "port",
				Usage: "Port for this node to listen on.",
				Value: 52386,
			},
			&cli.StringFlag{
				Name:  "remote-address",
				Usage: "Destination server address to seed peer discovery.",
			},
			&cli.UintFlag{
				Name:  "remote-port",
				Usage: "Port the peer daemon is listening on.",
				Value: 52386,
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug mode.",
			},
		},

		Action: actStartNode,
	}

	if err := app.Run(args); err != nil {
		return err
	}

	return nil
}

func actStartNode(c *cli.Context) error {
	var (
		// This gets the named variables from the context of the parsed args
		LocalAddress  = c.String("address")
		LocalPort     = c.Uint("port")
		RemoteAddress = c.String("remote-address")
		RemotePort    = c.Uint("remote-port")
		Debug         = c.Bool("debug")
	)
	// first set up the logger
	var logger *zap.Logger

	var err error
	if Debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Fprint(os.Stderr, "Unable to initialize zap logger. Exiting.")
		os.Exit(1)
	}
	defer logger.Sync()
	logger.Debug("Zap logger started in debug mode")

	// Verify local address is a valid address if specified
	var serverArgs ServerArgs
	nodeIP := net.ParseIP(LocalAddress)
	if nodeIP == nil && LocalAddress != "" {
		return fmt.Errorf("%s is not a valid ip address", LocalAddress)
	}
	serverArgs.NodeOpts = append(serverArgs.NodeOpts, noise.WithNodeBindHost(nodeIP))

	// Verify local port is a valid port
	if LocalPort == 0 || LocalPort > 65535 {
		return fmt.Errorf("%d is not a valid port number (1-65535)", LocalPort)
	}
	serverArgs.NodeOpts = append(serverArgs.NodeOpts, noise.WithNodeBindPort(uint16(LocalPort)))

	// Verify peer port is a valid port
	if RemotePort == 0 || RemotePort > 65535 {
		return fmt.Errorf("%d is not a valid port number (1-65535)", RemotePort)
	}

	// parse peer IP address
	peerIP := net.ParseIP(RemoteAddress)
	if peerIP == nil && RemoteAddress != "" {
		addr, err := net.LookupIP(RemoteAddress)
		if err != nil {
			return fmt.Errorf(`"%s" does not appear to be a valid ip address or server name`, RemoteAddress)
		}
		fmt.Fprintf(c.App.ErrWriter, "Peer %s resolves to %s\n", RemoteAddress, addr[0].String())
	}
	serverArgs.PeerAddress = fmt.Sprintf("%s:%d", RemoteAddress, RemotePort)
	serverArgs.Logger = logger

	var me ServerNode

	err = me.Init(serverArgs)
	if err != nil {
		return err
	}

	// start the server listening
	if err = me.Listen(); err != nil {
		return err
	}
	// run discovery and see who you find
	me.StartDiscovery(1)
	waiter.WaitForSignal(os.Interrupt)
	me.StopDiscovery()
	fmt.Println("Exiting.")
	return nil
}
