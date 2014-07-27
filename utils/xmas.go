package main

import (
  "flag"
  "fmt"
  "os"
  "time"
  "github.com/adammck/hexapod"
)

var (
  portName = flag.String("port", "/dev/ttyACM0", "the serial port path")
  interval = flag.Int("interval", 50, "the time between steps (ms)")
  debug = flag.Bool("debug", false, "show serial traffic")
)

func main() {
  flag.Parse()

  h, err := hexapod.NewHexapodFromPortName(*portName)
  if err != nil {
    fmt.Printf("Error creating hexapod: %s\n", err)
    os.Exit(1)
  }

  h.Network.Debug = *debug
  led := true

  for {
    for _, leg := range h.Legs {
      for _, servo := range leg.Servos() {
        err := servo.SetLed(led)
        if err != nil {
          fmt.Printf("Error switching led on servo %d: %s\n", servo.Ident, err)
          os.Exit(1)
        }
      }
    }

    time.Sleep(time.Duration(*interval) * time.Millisecond)
    led = !led
  }
}
