# Adam's Junky Hexapod

This is the Go program which powers my crappy hexapod. I don't imagine that it's
very good, but I've learned a lot while building and writing it. Here's a model:

![hexapod sketchup model] (http://cl.ly/image/0c3p3h2R0N1R/hexapod-sketchup-20141125.png)


## Hardware

* [Printrbot Simple Metal] (http://printrbot.com/shop/assembled-simple-metal/)
* [Dynamixel AX-12A] (http://www.trossenrobotics.com/dynamixel-ax-12-robot-actuator.aspx)
* [Raspberry Pi, Model B+] (http://www.raspberrypi.org/products/model-b-plus/)
* [USB2AX] (http://www.xevelabs.com/doku.php?id=product:usb2ax:usb2ax)
* [SparkFun USB MicroB Breakout] (https://www.sparkfun.com/products/10031)
* [USB Super RetroPort] (http://www.retrousb.com/product_info.php?cPath=21&products_id=29)
* [RioRand LM2596] (http://amzn.com/B008BHAOQO)
* [Tiger 11.1v 2200mAh LiPo] (http://www.trossenrobotics.com/3s-11v-2200mah-25c-lipo-battery)
* [Mean Well GS90A12-P1M] (http://www.jameco.com/webapp/wcs/stores/servlet/Product_10001_10001_2078291_-1) power supply
* [Edimax EW-7811Un] (http://amzn.com/B003MTTJOY)
* [Medialink MUA-BA3] (http://amzn.com/B004LNXO28) bluetooth adapter


## Usage

To build and deploy a release:

    bin/pi-deploy main/bot.go

To power down:

    bin/pi-shutdown
