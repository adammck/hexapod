package hexapod

type Point3d struct {
	X float64
	Y float64
	Z float64
}

func NewPoint(X float64, Y float64, Z float64) *Point3d {
	return &Point3d{
		X: X,
		Y: Y,
		Z: Z,
	}
}
