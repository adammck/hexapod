package hexapod

import (
	"fmt"
)

type Segment struct {
	Name     string
	parent   *Segment
	Child    *Segment
	Pair     Pair
	Angle    EulerAngles
	//eaStart  EulerAngles
	//eaEnd    EulerAngles
	vec      Vector3
	locked   bool
	eaLock   EulerAngles
}

func MakeSegment(name string, parent *Segment, pair Pair, vec Vector3) *Segment {

	//eaStart := pair.one
	//eaEnd := pair.two

	s := &Segment{
		Name:    name,
		parent:  parent,
		Pair:    pair,
		Angle:   pair.one,
		//eaStart: eaStart,
		//eaEnd:   eaEnd,
		vec:     vec,
	}

	if parent != nil {
		parent.Child = s
	}

	return s
}

func MakeRootSegment(vec Vector3) *Segment {
	return MakeSegment("root", nil, Pair{}, vec)
}

func (s Segment) String() string {
	var childStr string

	if s.Child != nil {
		childStr = s.Child.String()
	} else {
		childStr = "nil"
	}

	return fmt.Sprintf("&Seg{%s: %s %s}", s.Name, s.Angle, childStr)
}



func (s *Segment) Clone() *Segment {

	var c *Segment
	if s.Child != nil {
		c = s.Child.Clone()
	}

	ss := &Segment{
		Name:    s.Name,
		parent:  s.parent,
		Child:   c,
		Pair:    s.Pair,
		Angle:   s.Angle,
		vec:     s.vec,
		locked:  s.locked,
		eaLock:  s.eaLock,
	}

	if c != nil {
		c.parent = ss
	}

	return ss
}

func (s *Segment) EulerAngles(resolution float64) []EulerAngles {
	if s.locked {
		return []EulerAngles{s.eaLock}
	}

	return s.Pair.EulerAngles(resolution)
}



// func (s *Segment) Range(step float64) []EulerAngles {
//   if(s.locked) {
//     ea = append(ea, s.eaLock)
//     return ea
//   }
// }

func (s *Segment) LockRotation(ea EulerAngles) {
	s.Angle = ea
	s.eaLock = ea
	s.locked = true
}

func (s *Segment) Unlock() {
	s.Angle = IdentityOrientation
	s.eaLock = IdentityOrientation
	s.locked = false
}

// TODO: GTFO, do this in range, don't mutate it from outside.
func (s *Segment) SetRotation(r EulerAngles) {
	s.Angle = r
}

// Start returns a vector3 with the coordinates of the start of this segment, in
// the world coordiante space.
func (s *Segment) Start() Vector3 {
	return s.Project(ZeroVector3)
}

// Start returns a vector3 with the coordinates of the end of this segment, in
// the world coordiante space.
func (s *Segment) End() Vector3 {
	return s.Project(s.vec)
}

// WorldMatrix returns a Matrix4 which can be applied to a vector in this
// segment's coordinate space to convert it to the world space.
func (s *Segment) WorldMatrix() *Matrix44 {

	// if this segment has a parent, our transformation will start at the zero of
	// that space, move by the vector (to the end of the segment), and rotate into
	// this coordinate space.
	if s.parent != nil {
		m := MakeMatrix44(s.parent.vec, s.Angle)
		return MultiplyMatrices(*m, *s.parent.WorldMatrix())

		// no parent means that this is a root segment, so the origin is zero, and
		// transformations only need an angle.
	} else {
		return MakeMatrix44(ZeroVector3, s.Angle)
	}
}

// Project transforms a vector in this segment's coordinate space into a vector3
// in the world space.
// (pointer to a) new vector in the world space.
func (s *Segment) Project(v Vector3) Vector3 {
	return v.MultiplyByMatrix44(*s.WorldMatrix())
}
