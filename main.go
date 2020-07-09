package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"

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
			&cli.PathFlag{
				Name:      "load-private-key",
				Aliases:   []string{"loadkey", "l"},
				Usage:     "Specify private key. Otherwise, noise will generate a new one.",
				TakesFile: true,
			},
			&cli.PathFlag{
				Name:      "make-private-key",
				Aliases:   []string{"mk"},
				Usage:     "Creates an ed25519 private key at the specified path and exits.",
				TakesFile: true,
			},
			&cli.IntFlag{
				Name:    "discovery-interval",
				Aliases: []string{"di"},
				Usage:   "Configures the interval, in seconds, between peer discovery attempts.",
				Value:   5,
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
		LocalAddress      = c.String("address")
		LocalPort         = c.Uint("port")
		RemoteAddress     = c.String("remote-address")
		RemotePort        = c.Uint("remote-port")
		Debug             = c.Bool("debug")
		KeyPath           = c.Path("load-private-key")
		MakeKeypath       = c.Path("make-private-key")
		DiscoveryInterval = c.Int("discovery-interval")
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

	// if we're asked to make a key, do that and exit
	if MakeKeypath != "" {
		err := makeNewKeys(MakeKeypath)
		if err != nil {
			return err
		}
		fmt.Printf("new ed25519 private key written to %s\nExiting\n", MakeKeypath)
		os.Exit(0)
	}

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

	// set the self-explantory options
	serverArgs.Logger = logger
	serverArgs.DiscoveryInterval = DiscoveryInterval

	// if the user set a private key, load it
	if KeyPath != "" {
		if _, err := os.Stat(KeyPath); !os.IsNotExist(err) {
			// path/to/whatever exists
			data, err := ioutil.ReadFile(KeyPath)
			if err != nil {
				return errors.New("Unable to read private key file")
			}
			// now convert to a noise private key
			noiseKey, err := noise.LoadKeysFromHex(string(data))
			if err != nil {
				return fmt.Errorf("noise is unable to load the hex key from file: %+v", err)
			}
			logger.Sugar().Infow("successfully loaded private key",
				"publickey", noiseKey.Public().String(),
			)
			serverArgs.NodeOpts = append(serverArgs.NodeOpts, noise.WithNodePrivateKey(noiseKey))
		}
	}

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
	me.StartDiscovery()

	go func(s *ServerNode) {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case _ = <-ticker.C:
				s.Hub.Push(context.TODO(), []byte("hello"))
				s.Log.Debug("sent test gossip")
			}
		}
	}(&me)
	waiter.WaitForSignal(os.Interrupt)
	me.StopDiscovery()
	fmt.Println("Exiting.")
	return nil
}

func makeNewKeys(path string) error {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}
	data := make([]byte, hex.EncodedLen(len(priv)))
	hex.Encode(data, priv)
	err = ioutil.WriteFile(path, data, 0600)
	if err != nil {
		return err
	}
	return nil
}
