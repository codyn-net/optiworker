package main

import (
	"github.com/jessevdk/go-configure"
)

func main() {
	configure.Version = []int {3, 0}
	configure.Target = "optiworker"

	configure.Configure(nil)
}
