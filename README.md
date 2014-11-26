# Adam's Junky Hexapod

This is the Go program which powers my crappy hexapod. I don't imagine that it's very good, but I've learned a lot while building and writing it. Here's a model:

![hexapod sketchup model] (http://cl.ly/image/0c3p3h2R0N1R/hexapod-sketchup-20141125.png)


## Hardware

* [Printrbot Simple Metal] (http://printrbot.com/shop/assembled-simple-metal/)
* [Dynamixel AX-12A] (http://www.trossenrobotics.com/dynamixel-ax-12-robot-actuator.aspx)
* [Raspberry Pi, Model B+] (http://www.raspberrypi.org/products/model-b-plus/)
* [USB2AX] (http://www.xevelabs.com/doku.php?id=product:usb2ax:usb2ax)
* [USB Super RetroPort] (http://www.retrousb.com/product_info.php?cPath=21&products_id=29)
* [RioRand LM2596] (http://amzn.com/B008BHAOQO)

Total build cost (excluding the printer), is about $800, many evenings and weekends, and repeatedly lacerated hands.


## Requirements

The hexapod's hostname must be `hexapod`.


## Usage

To deploy and run an arbitrary package:

    bin/pi-run utils/xmas.go

To power down:

    bin/pi-shutdown
