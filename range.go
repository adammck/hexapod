package hexapod

import (
  "math"
  "fmt"
)

type Pair struct {
  one EulerAngles
  two EulerAngles
}

const (
  RotationHeading rotation = iota
  RotationPitch   rotation = iota
  RotationBank    rotation = iota
)

func MakePair(rot rotation, degOne float64, degTwo float64) *Pair {
  return &Pair{
    *MakeSingularEulerAngle(rot, degOne),
    *MakeSingularEulerAngle(rot, degTwo),
  }
}

func (p Pair) String() string {
  return fmt.Sprintf("&Pair{%s %s}", p.one, p.two)
}

func (p Pair) EulerAngles(resolution float64) []EulerAngles {
  ea := make([]EulerAngles, 0)

  minX := math.Min(p.one.Heading, p.two.Heading)
  maxX := math.Max(p.one.Heading, p.two.Heading)
  stpX := (maxX - minX) * resolution

  minY := math.Min(p.one.Pitch, p.two.Pitch)
  maxY := math.Max(p.one.Pitch, p.two.Pitch)
  stpY := (maxY - minY) * resolution
  //

  minZ := math.Min(p.one.Bank, p.two.Bank)
  maxZ := math.Max(p.one.Bank, p.two.Bank)
  stpZ := (maxZ - minZ) * resolution
  //

  if stpX == 0 {
    stpX = 1
  } else {
    //fmt.Printf("minX=%0.4f, maxX=%0.4f, stepX=%0.4f\n", minX, maxX, stpX)
  }
  if stpY == 0 {
    stpY = 1
  } else {
    //fmt.Printf("minY=%0.4f, maxY=%0.4f, stepY=%0.4f\n", minY, maxY, stpY)
  }
  if stpZ == 0 {
    stpZ = 1
  } else {
    //fmt.Printf("minZ=%0.4f, maxZ=%0.4f, stepZ=%0.4f\n", minZ, maxZ, stpZ)
  }

  for x := minX; x <= maxX; x += stpX {
    for y := minY; y <= maxY; y += stpY {
      for z := minZ; z <= maxZ; z += stpZ {
        ea = append(ea, *MakeEulerAngles(x, y, z))
        //fmt.Printf("%0.4f, %0.4f, %0.4f\n", x, y, z)
      }
    }
  }

  //fmt.Println(len(ea))
  return ea
}

// func (p Pair) EulerAngles(resolution float64) []EulerAngles {
//   ea := make([]EulerAngles, 0)

//   minX := math.Min(p.one.Heading, p.two.Heading)
//   maxX := math.Max(p.one.Heading, p.two.Heading)
//   minY := math.Min(p.one.Pitch, p.two.Pitch)
//   maxY := math.Max(p.one.Pitch, p.two.Pitch)
//   minZ := math.Min(p.one.Bank, p.two.Bank)
//   maxZ := math.Max(p.one.Bank, p.two.Bank)

//   for x := minX; x <= maxX; x += resolution {
//     for y := minY; y <= maxY; y += resolution {
//       for z := minZ; z <= maxZ; z += resolution {
//         ea = append(ea, *MakeEulerAngles(x, y, z))
//       }
//     }
//   }

//   return ea
// }

func (p Pair) Zoom(ea EulerAngles, factor float64) Pair {

  //fmt.Printf("Zooming: %v x%0.4f (%v)\n", ea, factor, p)

  h1, h2 := z(ea.Heading, p.one.Heading, p.two.Heading, factor)
  p1, p2 := z(ea.Pitch, p.one.Pitch, p.two.Pitch, factor)
  b1, b2 := z(ea.Bank, p.one.Bank, p.two.Bank, factor)

  one := EulerAngles{
    // Heading: mid(p.one.Heading, ea.Heading, factor),
    // Pitch:   mid(p.one.Pitch, ea.Pitch, factor),
    // Bank:    mid(p.one.Bank, ea.Bank, factor),
    Heading: h1,
    Pitch:   p1,
    Bank:    b1,
  }

  two := EulerAngles{
    // Heading: mid(p.two.Heading, ea.Heading, factor),
    // Pitch:   mid(p.two.Pitch, ea.Pitch, factor),
    // Bank:    mid(p.two.Bank, ea.Bank, factor),
    Heading: h2,
    Pitch:   p2,
    Bank:    b2,
  }

  return Pair{one, two}

}

func z(mid float64, one float64, two float64, f float64) (float64, float64) {

  min := math.Min(one, two)
  max := math.Max(one, two)

  if mid < min || mid > max {
    fmt.Printf("mid: %0.4f, min: %0.4f, max: %0.4f\n", deg(mid), deg(min), deg(max))
    panic("nope!")
  }

  r := ((max - min) * f) * 0.5
  newMin := (mid - r)
  newMax := (mid + r)

  //fmt.Printf("Z! 1=%0.4f 2=%0.4f r=%0.4f, 1=%0.4f 2=%0.4f\n", deg(one), deg(two), r, deg(mid - r), deg(mid + r))
  return newMin, newMax
}

// Given three EulerAngles (mid, start, end), returns two new (start, end)
// angles which are closer to mid.
// TODO: Encapsulate angle pairs in a Range struct
// func NarrowRange(angle EulerAngles, start EulerAngles, end EulerAngles, factor float64) (EulerAngles, EulerAngles) {
//   s := EulerAngles{
//     Heading: mid(start.Heading, angle.Heading, factor),
//     Pitch:   mid(start.Pitch, angle.Pitch, factor),
//     Bank:    mid(start.Bank, angle.Bank, factor),
//   }

//   e := EulerAngles{
//     Heading: mid(end.Heading, angle.Heading, factor),
//     Pitch:   mid(end.Pitch, angle.Pitch, factor),
//     Bank:    mid(end.Bank, angle.Bank, factor),
//   }

//   return s, e
// }

func mid(a float64, b float64, f float64) float64 {
  return a + ((b - a) * f)
}
