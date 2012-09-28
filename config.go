package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"ponyo.epfl.ch/go/get/optimization/go/optimization/constants"
)

type Config struct {
	ProtocolVersion uint

	DiscoveryNamespace string `short:"d" long:"discovery-namespace" description:"The discovery namespace"`
	DiscoveryAddress   string `long:"discovery-address" description:"The discovery address"`

	ListenAddress string `short:"l" long:"listen-address" description:"The listen address"`

	UseAuthentication  bool `long:"use-authentication" description:"Whether or not to use authentication"`
	DispatcherPriority uint `long:"dispatcher-priority" description:"The dispatcher process priority"`

	Parallel int `short:"p" long:"parallel" description:"The number of workers to run in parallel. Zero or negative values indicate to use the number of available CPUs minus the specified value"`
}

func (c *Config) Load(filename string) {
	f, err := os.Open(filename)

	if err != nil {
		return
	}

	dec := json.NewDecoder(f)
	dec.Decode(c)
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

		ListenAddress:      ":0",
		UseAuthentication:  false,
		Parallel:           1,
		DispatcherPriority: 0,
	}
}
