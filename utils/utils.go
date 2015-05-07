package utils

import (
	"math"
)

func Deg(rads float64) float64 {
	return rads / (math.Pi / 180)
}

func Rad(degrees float64) float64 {
	return (math.Pi / 180) * degrees
}

func sign(n float64) float64 {
	if n > 0 {
		return 1.0

	} else if n < 0 {
		return -1.0

	} else {
		return 0.0
	}
}
