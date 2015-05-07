package main

import (
	"flag"
	"fmt"
	"github.com/adammck/dynamixel"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/components/controller"
	"github.com/adammck/hexapod/components/legs"
	"github.com/jacobsa/go-serial/serial"
	//"github.com/adammck/hexapod/components/voltage"
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

	fmt.Println("Opening controller...")
	f, err := os.Open("/dev/input/event0")
	if err != nil {
		fmt.Printf("error opening controller: %s\n", err)
		os.Exit(1)
	}

	network := dynamixel.NewNetwork(serial)
	network.Debug = *debug
	h := hexapod.NewHexapod(network)

	fmt.Println("Creating components...")
	h.Add(legs.New(h, network))
	//h.Add(voltage.New())
	h.Add(controller.New(h, f))

	fmt.Println("Booting components...")
	h.Boot()

	fmt.Println("Initializing loop...")
	t := time.NewTicker(1 * time.Second / 60)

	// Catch both SIGINT (ctrl+c) and SIGTERM (kill/systemd), to allow the hexapod
	// to power down its servos before exiting.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range c {
			fmt.Println("Halting")
			//h.Halt = true

			// Keep looping for three seconds before quitting, to give everything time
			// to shut down gracefully.
			time.Sleep(3 * time.Second)
			t.Stop()
			os.Exit(2)
		}
	}()

	// Run until START (bounce service) or SELECT+START (poweroff).
	fmt.Println("Starting loop...")
	for now := range t.C {
		h.Tick(now)
	}

	// code := h.MainLoop()
	// if code == 1 {
	// 	fmt.Println("Shutting down")
	// 	cmd := exec.Command("poweroff")
	// 	err := cmd.Run()
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// }

	// fmt.Printf("Exit(%d)\n", code)
	// os.Exit(code)
}
