package main

import (
	"fmt"
	"os"

	"github.com/haukened/waiter"
	"github.com/sirupsen/logrus"
)

func init() {
	var log = logrus.New()
	log.Formatter = &logrus.TextFormatter{
		DisableColors: true,
	}
	log.SetOutput(os.Stdout)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// parse the application cli args
	args, err := parseArgs()
	if err != nil {
		return err
	}
	// create a new instance of ServerNode and set it up
	var me ServerNode
	err = me.Init(*args)
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
