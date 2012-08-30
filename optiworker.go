package main

import (
	"./config"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"ponyo.epfl.ch/go/get/optimization/go/optimization"
	"runtime"
	"syscall"
)

var _ = fmt.Println

var TheConfig *Config
var apps []*App

func setupApps() {
	var num int

	if TheConfig.Parallel <= 0 {
		num = runtime.NumCPU() + TheConfig.Parallel
	} else {
		num = TheConfig.Parallel
	}

	if num <= 0 {
		num = 1
	}

	for i := 0; i < num; i++ {
		apps = append(apps, NewApp(uint(i)))
	}
}

func main() {
	TheConfig = NewConfig()

	TheConfig.Load(path.Join(config.Sysconfdir, "optiworker", "config.json"))
	TheConfig.SetupParse()

	flag.Parse()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		_ = <-c

		// Either way, try to terminate cleanly
		for _, app := range apps {
			app.Close()
		}

		os.Exit(1)
	}()

	setupApps()

	optimization.Events.Loop()
}
