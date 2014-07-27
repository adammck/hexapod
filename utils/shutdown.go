package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/adammck/hexapod"
)

var (
	portName = flag.String("port", "/dev/ttyACM0", "the serial port path")
	debug = flag.Bool("debug", false, "show serial traffic")
)

func main() {
	flag.Parse()

	h, err := hexapod.NewHexapodFromPortName(*portName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	h.Network.Debug = *debug
	h.Shutdown()
}
