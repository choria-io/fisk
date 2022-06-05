package main

import (
	"fmt"

	"github.com/choria-io/fisk"
)

var (
	debug   = fisk.Flag("debug", "Enable debug mode.").Bool()
	timeout = fisk.Flag("timeout", "Timeout waiting for ping.").Envar("PING_TIMEOUT").Required().Short('t').Duration()
	ip      = fisk.Arg("ip", "IP address to ping.").Required().IP()
	count   = fisk.Arg("count", "Number of packets to send").Int()
)

func main() {
	fisk.Version("0.0.1")
	fisk.Parse()
	fmt.Printf("Would ping: %s with timeout %s and count %d", *ip, *timeout, *count)
}
