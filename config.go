package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"ponyo.epfl.ch/go/get/optimization/go/optimization/constants"
)

type Config struct {
	ProtocolVersion uint

	DiscoveryNamespace string
	DiscoveryAddress   string

	ListenAddress string

	UseAuthentication  bool
	DispatcherPriority uint

	Parallel int
}

func (c *Config) Load(filename string) {
	f, err := os.Open(filename)

	if err != nil {
		return
	}

	dec := json.NewDecoder(f)
	dec.Decode(c)
}

func (c *Config) SetupParse() {
	flag.StringVar(&c.DiscoveryNamespace,
		"discovery",
		c.DiscoveryNamespace,
		"Discovery namespace")

	flag.StringVar(&c.DiscoveryAddress,
		"discovery-address",
		c.DiscoveryAddress,
		"Discovery address")

	flag.StringVar(&c.ListenAddress,
		"listen-address",
		c.ListenAddress,
		"Listen address")

	flag.BoolVar(&c.UseAuthentication,
		"use-authentication",
		c.UseAuthentication,
		"Use authentication")

	flag.IntVar(&c.Parallel,
		"parallel",
		c.Parallel,
		"Run N workers in parallel (if the specified value <= 0 then number of parallel workers is value + NUM_CPUS)")

	flag.UintVar(&c.DispatcherPriority,
		"dispatcher-priority",
		c.DispatcherPriority,
		"Specify dispatcher priority (> 0)")
}

func NewConfig() *Config {
	us, _ := user.Current()

	return &Config{
		ProtocolVersion: 2,

		DiscoveryNamespace: func() string {
			if us == nil {
				return ""
			}

			return us.Username
		}(),

		DiscoveryAddress: fmt.Sprintf("%v:%v",
			constants.DiscoveryGroup,
			constants.DiscoveryPort),

		ListenAddress:      fmt.Sprintf(":%v", constants.WorkerPort),
		UseAuthentication:  false,
		Parallel:           1,
		DispatcherPriority: 0,
	}
}
