# Adam's Hexapod

This is the Go program which powers my hexapod. It runs on a Raspberry Pi B+, and is controlled by a Sony Sixaxis (PS3) controller. The hexapod itself is 50cm in diameter, and about 2kg. The chassis was 3d-printed with a Printrbot Simple Metal, and bolted to 24 Dynamixel AX-12A servos. Each leg has 4DOF, which makes the gait quite flexible.

Here's it is, as of January 2015:

![hexapod photo](http://i.imgur.com/h3j1Ojn.jpg)

And here's a GIF of it in action:

![hexapod standing up](https://i.imgur.com/YVEN3If.gif)

[More GIFs here](https://imgur.com/a/eXqIa).  
And [here's a video](https://vimeo.com/115932070).


## Hardware

* [Printrbot Simple Metal](http://printrbot.com/shop/assembled-simple-metal/) 3d printer
* [Dynamixel AX-12A](http://www.trossenrobotics.com/dynamixel-ax-12-robot-actuator.aspx) servos
* [Raspberry Pi, Model B+](http://www.raspberrypi.org/products/model-b-plus/) computer
* [Sony Sixaxis](https://en.wikipedia.org/wiki/Sixaxis) controller
* [USB2AX](http://www.xevelabs.com/doku.php?id=product:usb2ax:usb2ax) dynamixel interface
* [SparkFun USB MicroB Breakout](https://www.sparkfun.com/products/10031)
* [RioRand LM2596](http://amzn.com/B008BHAOQO) voltage regulator
* [Kootek GY-521](http://a.co/0PT1ztv) gyro/accelerometer
* [Tiger 11.1v 2200mAh LiPo](http://www.trossenrobotics.com/3s-11v-2200mah-25c-lipo-battery) battery
* [Corsair RM450](http://a.co/aM8V92D) power supply
* [Edimax EW-7811Un](http://amzn.com/B003MTTJOY) wifi adapter
* [Medialink MUA-BA3](http://amzn.com/B004LNXO28) bluetooth adapter
* [JW Winco 353-20-24-M6-S-55](https://store.solutionsdirectonline.com/jw-winco-353-20-24-m6-s-55-vibration-isolation-mount-p9026.aspx) rubber feet
* [Microsoft LifeCam Cinema 720p](http://a.co/3BNVShU) camera

## Usage

1. Spend countless hours and dollars printing and assembling the hexapod. Be
   sure to blow up RPi and trap fingers between moving parts for authentic
   experience.

2. Provision the RPi using
   [adammck/headless-raspbian](https://github.com/adammck/headless-raspbian)
   and
   [adammck/hexapod-infra](https://github.com/adammck/hexapod-infra).
   It runs Raspbian Jessie with
   [QtSixa](http://qtsixa.sourceforge.net),
   the control program (this repo), and a few systemd services to glue
   everything together.

3. Flip the power switch to boot the hexapod. If you're running tethered with an
   external PSU, make sure that the power switch is off to isolate the LiPo
   before plugging it in.

4. Plug the Sixaxis controller in with a USB cable. You only have to do this
   once, to pair it with the Bluetooth adaptor. Give it a few seconds (to run
   sixpair), then unplug it.

5. Build and deploy:

        bin/pi-deploy main/main.go

   This requires Go to be installed with cross-compilation support for
   Linux/ARM. That's outside of the scope of this document, but it's easy.

6. Press the PS button to pair. The controller should rumble and flash its
   lights. The control program will now start, and the hexapod will initialize
   and stand up.

7. Use the left stick to translate, and L2/R2 to rotate. Various other buttons
   do other things.

8. Press Select and Start to shut down the servos and the RPi. Note that this
   doesn't entirely kill the power, so don't forget to disconnect the LiPo to
   avoid damaging it.

   Shut down remotely by running:

        bin/pi-poweroff

    Shutdown will automatically occur (with no warning) when the battery drops
    below 9.6 volts. This is to protect the LiPo. My 2200mAh battery usually
    lasts about 15 minutes on a full charge.


## License

MIT
