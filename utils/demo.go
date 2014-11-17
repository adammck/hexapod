package main

import (
  "flag"
  "fmt"
  "os"
  "os/signal"
  "syscall"
  "os/exec"
  "runtime/pprof"
  "github.com/adammck/hexapod"
  "github.com/adammck/retroport"
)

var (
  portName = flag.String("port", "/dev/ttyACM0", "the serial port path")
  debug = flag.Bool("debug", false, "show serial traffic")
  cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

func main() {
  flag.Parse()

  if *cpuprofile != "" {
    f, err := os.Create(*cpuprofile)
    if err != nil {
      fmt.Println(err)
      os.Exit(1)
    }
    pprof.StartCPUProfile(f)
  }

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

  // Catch both SIGINT (ctrl+c) and SIGTERM (kill/systemd), to allow the hexapod
  // to power down its servos before exiting.
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt, syscall.SIGTERM)
  go func() {
    for _ = range c {
      fmt.Println("Halting")
      h.Halt = true
    }
  }()

  // Run until START (bounce service) or SELECT+START (poweroff).
  code := h.MainLoop()
  if code == 1 {
    fmt.Println("Shutting down")
    cmd := exec.Command("poweroff")
    err := cmd.Run()
    if err != nil {
      fmt.Println(err)
    }
  }

  if *cpuprofile != "" {
    pprof.StopCPUProfile()
  }

  fmt.Printf("Exit(%d)\n", code)
  os.Exit(code)
}
