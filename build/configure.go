package main

import (
	"github.com/jessevdk/go-configure"
)

func main() {
	configure.Version = []int {2, 12}
	configure.Target = "optiworker"

	configure.Configure(nil)
}