package game

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	// "github.com/runningwild/magnus/stats"
)

type Snare struct {
	BaseEnt
	NonManaUser
}

func init() {
	gob.Register(&Snare{})
}

func (s *Snare) Copy() Ent {
	s2 := *s
	return &s2
}

func (s *Snare) Draw(g *Game) {
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4ub(255, 255, 255, 255)
	p := s.Pos()
	x := gl.Int(p.X)
	y := gl.Int(p.Y)
	gl.Begin(gl.LINES)
	gl.Vertex2i(x-12, y-12)
	gl.Vertex2i(x-12, y+12)
	gl.Vertex2i(x-12, y+12)
	gl.Vertex2i(x+12, y+12)
	gl.Vertex2i(x+12, y+12)
	gl.Vertex2i(x+12, y-12)
	gl.Vertex2i(x+12, y-12)
	gl.Vertex2i(x-12, y-12)
	gl.End()
}
func (s *Snare) Alive() bool {
	return s.Stats.HealthCur() > 0
}
func (s *Snare) OnDeath(g *Game) {
}
func (s *Snare) Think(g *Game) {
	s.BaseEnt.Think(g)
}
