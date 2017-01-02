package controller

type Latch struct {
	val bool
}

func (l *Latch) Run(v bool) bool {
	r := v && !l.val
	l.val = v
	return r
}
