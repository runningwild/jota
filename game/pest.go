package game

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
)

type Pest struct {
	BaseEnt
	NonManaUser
}

func init() {
	gob.Register(&Pest{})
}

func (p *Pest) Copy() Ent {
	p2 := *p
	return &p2
}

func (p *Pest) Draw(g *Game) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.01)
	base.SetUniformF("status_bar", "outer", 0.03)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(255, 255, 255, 255)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)
	base.SetUniformF("status_bar", "inner", 0.04)
	base.SetUniformF("status_bar", "outer", 0.045)
	base.SetUniformF("status_bar", "buffer", 0.01)
	health_frac := float32(p.Stats.HealthCur() / p.Stats.HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, 255)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, 255)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)
	base.EnableShader("")
}
func (p *Pest) Alive() bool {
	return p.Stats.HealthCur() > 0
}
func (p *Pest) OnDeath(g *Game) {
	for _, ent := range g.Ents {
		d := ent.Pos().Sub(p.Pos()).Mag2()
		if d < 100*100 {
			ent.ApplyDamage(stats.Damage{stats.DamageFire, 400})
		}
	}
}
func (p *Pest) Think(g *Game) {
	p.BaseEnt.Think(g)
	var target Ent
	dist := 1.0e9
	for _, ent := range g.Ents {
		player, ok := ent.(*Player)
		if !ok {
			continue
		}
		d := player.Pos().Sub(p.Pos()).Mag2()
		if d < dist {
			dist = d
			target = player
		}
	}
	if target == nil {
		return
	}
	if dist < 50*50 {
		p.ApplyDamage(stats.Damage{stats.DamageFire, 1})
	}
	dir := target.Pos().Sub(p.Pos()).Norm().Scale(1.0)
	p.ApplyForce(dir.Scale(10.0))
}
