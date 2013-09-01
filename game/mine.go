package game

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
)

// Moba base ent
type Mine struct {
	BaseEnt
	Damage  float64
	Trigger float64
}

func (g *Game) MakeMine(pos linear.Vec2, health, mass, damage, trigger float64) {
	mine := Mine{
		BaseEnt: BaseEnt{
			Side_:        10,
			CurrentLevel: GidInvadersStart,
			Position:     pos,
		},
		Damage:  damage,
		Trigger: trigger,
	}
	mine.BaseEnt.StatsInst = stats.Make(stats.Base{
		Health: health,
		Mass:   mass,
		Size:   5,
	})
	g.AddEnt(&mine)
}

func (m *Mine) Think(g *Game) {
	m.BaseEnt.Think(g)
	prox := 50.0
	for _, ent := range g.temp.AllEnts {
		if ent == m {
			continue
		}
		if ent.Pos().Sub(m.Position).Mag2() < prox*prox {
			m.Trigger -= ent.Vel().Sub(m.Velocity).Mag2()
		}
	}
	if m.Trigger <= 0 {
		for _, ent := range g.temp.AllEnts {
			if ent.Pos().Sub(m.Position).Mag() < prox {
				ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, m.Damage})
			}
		}
	}
}

func (m *Mine) Draw(g *Game, side int) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.01)
	base.SetUniformF("status_bar", "outer", 0.03)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(255, 255, 255, 255)
	texture.Render(m.Position.X-100, m.Position.Y-100, 200, 200)
	base.SetUniformF("status_bar", "inner", 0.04)
	base.SetUniformF("status_bar", "outer", 0.045)
	base.SetUniformF("status_bar", "buffer", 0.01)
	health_frac := float32(m.Stats().HealthCur() / m.Stats().HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, 255)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, 255)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(m.Position.X-100, m.Position.Y-100, 200, 200)
	base.EnableShader("")
}
func (m *Mine) Supply(mana Mana) Mana { return Mana{} }
func (m *Mine) Walls() [][]linear.Vec2 {
	return nil
}
