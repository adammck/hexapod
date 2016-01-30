package main

import (
	"flag"
	"fmt"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/components/controller"
	"github.com/adammck/hexapod/components/legs"
	"github.com/adammck/hexapod/components/voltage"
	"github.com/jacobsa/go-serial/serial"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	portName = flag.String("port", "/dev/ttyACM0", "the serial port path")
	debug    = flag.Bool("debug", false, "show serial traffic")
)

func main() {
	flag.Parse()

	sOpts := serial.OpenOptions{
		PortName:              *portName,
		BaudRate:              1000000,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
	}

	fmt.Println("Opening serial port...")
	serial, err := serial.Open(sOpts)
	if err != nil {
		fmt.Printf("error opening serial port: %s\n", err)
		os.Exit(1)
	}
	defer serial.Close()

	fmt.Println("Purging serial buffer...")
	_, err = ioutil.ReadAll(serial)
	if err != nil {
		fmt.Printf("error purging serial buffer: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Opening controller...")
	f, err := os.Open("/dev/input/event0")
	if err != nil {
		fmt.Printf("error opening controller: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	network := network.New(serial)
	network.Debug = *debug
	h := hexapod.NewHexapod(network)

	fmt.Println("Creating components...")
	l := legs.New(network)
	h.Add(l)
	h.Add(voltage.New(l.Legs[0].Coxa))
	h.Add(controller.New(f))

	fmt.Println("Booting components...")
	err = h.Boot()
	if err != nil {
		fmt.Printf("error while booting: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Initializing loop...")
	t := time.NewTicker(1 * time.Second / 60)

	// Catch both SIGINT (ctrl+c) and SIGTERM (kill/systemd), to allow the hexapod
	// to power down its servos before exiting.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range c {
			fmt.Println("Caught signal, shutting down...")
			h.State.Shutdown = true
		}
	}()

	// Wait until h.Shutdown is true, then keep looping for three seconds, to give
	// everything time to shut down gracefully. Then quit.
	go func() {
		for {
			if h.State.Shutdown {
				fmt.Println("Shutdown requested, waiting 3...")
				time.Sleep(3 * time.Second)
				t.Stop()

				fmt.Println("Done waiting, shutting down...")
				os.Exit(2)
			}

			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Run until START (bounce service) or SELECT+START (poweroff).
	fmt.Println("Starting loop...")
	for now := range t.C {
		err = h.Tick(now, h.State)
		if err != nil {
			panic(err)
		}
	}
}
