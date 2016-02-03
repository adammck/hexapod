package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/adammck/dynamixel/network"
	"github.com/adammck/hexapod"
	"github.com/adammck/hexapod/components/controller"
	"github.com/adammck/hexapod/components/legs"
	"github.com/adammck/hexapod/components/voltage"
	fake_serial "github.com/adammck/hexapod/fake/serial"
	fake_voltage "github.com/adammck/hexapod/fake/voltage"
	"github.com/jacobsa/go-serial/serial"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	portName = flag.String("port", "/dev/ttyACM0", "the serial port path")
	debug    = flag.Bool("debug", false, "enable verbose logging")
	offline  = flag.Bool("offline", false, "run in offline mode (with fake devices)")
)

func main() {
	flag.Parse()
	var err error

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	sOpts := serial.OpenOptions{
		PortName:              *portName,
		BaudRate:              1000000,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
	}

	var srl io.ReadWriteCloser
	if *offline {
		log.Warn("using fake serial port")
		srl = &fake_serial.FakeSerial{}

	} else {
		log.Info("opening serial port")
		srl, err := serial.Open(sOpts)
		if err != nil {
			log.Fatalf("error opening serial port: %s\n", err)
		}
		defer srl.Close()

		log.Info("purging serial buffer")
		_, err = ioutil.ReadAll(srl)
		if err != nil {
			log.Fatalf("error purging serial buffer: %s\n", err)
		}
	}

	network := network.New(srl)
	network.Timeout = 100 * time.Millisecond
	h := hexapod.NewHexapod(network)

	log.Info("creating components")
	l := legs.New(network)
	h.Add(l)

	var f *os.File
	if *offline {
		log.Warn("using fake controller")
		f, _ = os.Open("/dev/null")
		defer f.Close()

	} else {
		log.Info("opening controller")
		f, err = os.Open("/dev/input/event0")
		if err != nil {
			log.Fatalf("error opening controller: %s", err)
		}
		defer f.Close()
	}
	h.Add(controller.New(f))

	var v voltage.HasVoltage
	if *offline {
		log.Warn("using fake voltage check")
		v = fake_voltage.New(9.6)
	} else {
		v = l.Legs[0].Coxa
	}
	h.Add(voltage.New(v))

	log.Info("booting components")
	err = h.Boot()
	if err != nil {
		log.Fatalf("error while booting: %s", err)
	}

	log.Info("initializing loop")
	t := time.NewTicker(1 * time.Second / 60)

	// Catch both SIGINT (ctrl+c) and SIGTERM (kill/systemd), to allow the hexapod
	// to power down its servos before exiting.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for _ = range c {
			log.Warn("caught signal, shutting down...")
			h.State.Shutdown = true
		}
	}()

	// Wait until h.Shutdown is true, then keep looping for three seconds, to give
	// everything time to shut down gracefully. Then quit.
	go func() {
		for {
			if h.State.Shutdown {
				log.Warn("shutdown requested, waiting 3 seconds...")
				time.Sleep(3 * time.Second)
				t.Stop()

				log.Warn("done waiting, exiting")
				os.Exit(1)
			}

			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Run until START (bounce service) or SELECT+START (poweroff).
	log.Info("starting loop")
	for now := range t.C {
		err = h.Tick(now, h.State)
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}
}
