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
  speed = flag.Int("speed", 512, "the movement speed")
  interval = flag.Int("interval", 10, "the time between steps (ms)")
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

  // set reasonably slow move speed
  for _, leg := range h.Legs {
    for _, servo := range leg.Servos() {
      servo.SetMovingSpeed(*speed)
    }
  }

  left := h.Legs[5]
  right := h.Legs[2]

  // relax src legs
  relaxLeg(left)
  relaxLeg(right)

  // loop forever
  for {
    syncLeg(left, h.Legs[1], h.Legs[3]) // mid left  -> (front+back) right
    syncLeg(right, h.Legs[0], h.Legs[4]) // mid right -> (front+back) left
    time.Sleep(time.Duration(*interval) * time.Millisecond)
  }
}

func relaxLeg(leg *hexapod.Leg) {
  for _, servo := range leg.Servos() {
    servo.SetTorqueEnable(false)
  }
}

func syncLeg(src *hexapod.Leg, dest... *hexapod.Leg) {
  for i, servo := range src.Servos() {
    p, _ := servo.Position()
    //fmt.Printf("%d=%d\n", i, p)

    for _, d := range dest {
      d.Servos()[i].SetGoalPosition(int(p))
    }
  }
}
