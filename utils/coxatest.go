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
  //h.setMoveSpeed(128)

  for i, leg := range h.Legs {
    fmt.Println(i)
    leg.Coxa.MoveTo(-40)
    time.Sleep(500 * time.Millisecond)

    leg.Coxa.MoveTo(40)
    time.Sleep(500 * time.Millisecond)

    leg.Coxa.MoveTo(0)
    time.Sleep(500 * time.Millisecond)
    fmt.Println("x")
  }

  os.Exit(0)
}
