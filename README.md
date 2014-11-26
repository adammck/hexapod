# Adam's Junky Hexapod

This is the Go program which powers my crappy hexapod. The hardware is a
Raspberry Pi, a USB2AX, and a bunch of Dynamixel AX-12A servos bolted to a
3D-printed skeleton. I don't expect that any of this will be useful to anyone
else, at least for the time being. Here it is:

![hexapod sketchup model] (http://cl.ly/image/0c3p3h2R0N1R/hexapod-sketchup-20141125.png)

## Requirements

The hexapod's hostname must be `hexapod`.


## Usage

To deploy and run an arbitrary package:

    bin/pi-run utils/xmas.go

To power down:

    bin/pi-shutdown
