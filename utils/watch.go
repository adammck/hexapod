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
  leg      = flag.Int("leg", 2, "the leg index to watch")
  interval = flag.Int("interval", 1000, "the time between reads (ms)")
  debug    = flag.Bool("debug", false, "show serial traffic")
)

func main() {
  flag.Parse()

  h, err := hexapod.NewHexapodFromPortName(*portName)
  if err != nil {
    fmt.Printf("Error creating hexapod: %s\n", err)
    os.Exit(1)
  }

  h.Network.Debug = *debug
  leg := h.Legs[*leg]

  for _, servo := range leg.Servos() {
    servo.SetTorqueEnable(false)
  }

  for {
    c, _ := leg.Coxa.Angle()
    fmt.Printf("Coxa=%.2f\n", c)

    f, _ := leg.Femur.Angle()
    fmt.Printf("Femur=%.2f\n", f)

    t, _ := leg.Tibia.Angle()
    fmt.Printf("Tibia=%.2f\n", t)

    tt, _ := leg.Tarsus.Angle()
    fmt.Printf("Tarsus=%.2f\n", tt)

    leg.UpdateSegments(c, f, t, tt)
    fmt.Printf("End=%v\n", leg.End())

    fmt.Println()
    time.Sleep(time.Duration(*interval) * time.Millisecond)
  }
}
