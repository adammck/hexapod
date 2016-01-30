package utils

import (
	"github.com/adammck/dynamixel/network"
	"math"
)

func Deg(rads float64) float64 {
	return rads / (math.Pi / 180)
}

func Rad(degrees float64) float64 {
	return (math.Pi / 180) * degrees
}

// Sync runs the given function while the network is in buffered mode, then
// initiates any movements at once by sending ACTION.
func Sync(n *network.Network, f func()) {
	n.SetBuffered(true)
	f()
	n.SetBuffered(false)
	n.Action()
}
