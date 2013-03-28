package main

import (
	"fmt"
	"os"
	"ponyo.epfl.ch/go/get/optimization-go/optimization"
	"ponyo.epfl.ch/go/get/optimization-go/optimization/log"
)

var _ = fmt.Println

type App struct {
	Id uint

	discovery *optimization.Discovery
	dispatch  *Dispatch
}

func (x *App) sendGreeting() {
	x.discovery.SendGreeting(x.dispatch.ListenAddress())
}

func (x *App) onWakeup(disc *optimization.Discovered) {
	x.sendGreeting()
}

func (x *App) Close() {
	x.dispatch.Close()
}

func (x *App) setupDiscovery() {
	// Create the discovery server
	var err error

	x.discovery, err = optimization.NewDiscovery(TheConfig.DiscoveryAddress,
		TheConfig.DiscoveryNamespace)

	if err != nil {
		log.E("Failed to start discovery server: %v", err)
		os.Exit(1)
	}

	// Set greeting callback
	x.discovery.Wakeup = append(x.discovery.Wakeup, func(disc *optimization.Discovered) {
		x.onWakeup(disc)
	})

	x.sendGreeting()
}

func (x *App) setupDispatch() {
	var err error

	x.dispatch, err = NewDispatch(TheConfig.ListenAddress, x.Id)

	if err != nil {
		log.E("Failed to start dispatch server: %v", err)
		os.Exit(1)
	}
}

func NewApp(id uint) *App {
	ret := &App{
		Id: id,
	}

	ret.setupDispatch()
	ret.setupDiscovery()

	return ret
}
