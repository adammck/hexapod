package hexapod

import (
	"math"
)

func deg(rads float64) float64 {
	return rads / (math.Pi / 180)
}

func rad(degrees float64) float64 {
	return (math.Pi / 180) * degrees
}
