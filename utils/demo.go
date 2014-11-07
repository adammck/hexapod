package main

import (
  "flag"
  "fmt"
  "os"
  "github.com/adammck/hexapod"
  "github.com/adammck/retroport"
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

  // open and connect the controller
  f, err := os.Open("/dev/hidraw0")
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  h.Controller = retroport.MakeSNES(f)
  go h.Controller.Run()

  // start rotating!
  h.MainLoop()
  os.Exit(0)
}
