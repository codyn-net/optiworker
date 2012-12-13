package main

import (
	"github.com/jessevdk/go-configure"
)

func main() {
	configure.Version = []int {2, 10}
	configure.Target = "optiworker"

	configure.Configure(nil)
}
