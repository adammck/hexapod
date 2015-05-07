package legs

import (
	"fmt"
	"github.com/adammck/hexapod/math3d"
)

type Segment struct {
	Name   string
	parent *Segment
	Child  *Segment
	Angles math3d.EulerAngles
	vec    math3d.Vector3
}

func MakeSegment(name string, parent *Segment, angles math3d.EulerAngles, vec math3d.Vector3) *Segment {
	s := &Segment{
		Name:   name,
		parent: parent,
		Angles: angles,
		vec:    vec,
	}

	if parent != nil {
		parent.Child = s
	}

	return s
}

func MakeRootSegment(vec math3d.Vector3) *Segment {
	return MakeSegment("root", nil, math3d.EulerAngles{}, vec)
}

func (s Segment) String() string {
	var childStr string

	if s.Child != nil {
		childStr = s.Child.String()
	} else {
		childStr = "nil"
	}

	return fmt.Sprintf("&Seg{%s: %s %s}", s.Name, s.Angles, childStr)
}

// Start returns a vector3 with the coordinates of the start of this segment, in
// the world coordiante space.
func (s *Segment) Start() math3d.Vector3 {
	return s.Project(math3d.ZeroVector3)
}

// Start returns a vector3 with the coordinates of the end of this segment, in
// the world coordiante space.
func (s *Segment) End() math3d.Vector3 {
	return s.Project(s.vec)
}

// WorldMatrix returns a Matrix4 which can be applied to a vector in this
// segment's coordinate space to convert it to the world space.
func (s *Segment) WorldMatrix() *math3d.Matrix44 {

	// if this segment has a parent, our transformation will start at the zero of
	// that space, move by the vector (to the end of the segment), and rotate into
	// this coordinate space.
	if s.parent != nil {
		m := math3d.MakeMatrix44(s.parent.vec, s.Angles)
		return math3d.MultiplyMatrices(*m, *s.parent.WorldMatrix())

		// no parent means that this is a root segment, so the origin is zero, and
		// transformations only need an angle.
	} else {
		return math3d.MakeMatrix44(math3d.ZeroVector3, s.Angles)
	}
}

// Project transforms a vector in this segment's coordinate space into a vector3
// in the world space.
// (pointer to a) new vector in the world space.
func (s *Segment) Project(v math3d.Vector3) math3d.Vector3 {
	return v.MultiplyByMatrix44(*s.WorldMatrix())
}
