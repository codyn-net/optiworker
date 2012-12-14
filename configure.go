package main

import (
	"github.com/jessevdk/go-configure"
)

func main() {
	configure.Version = []int {2, 11}
	configure.Target = "optiworker"

	configure.Configure(nil)
}
