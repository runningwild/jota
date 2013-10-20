package game

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/jota/texture"
	"github.com/runningwild/linear"
)

// Moba base ent
type HeatSeeker struct {
	BaseEnt
	HeatSeekerParams
}

type BaseEntParams struct {
	Health float64
	Mass   float64
	Size   float64
	Acc    float64
}

type ConditionMaker struct {
	Name   string
	Params map[string]int
}

type HeatSeekerParams struct {
	TargetGid Gid

	// The damage to do to ents in the AoE
	Damages []stats.Damage

	// The specs for the conditions to apply to players in the aoe.
	ConditionMakers []ConditionMaker

	// How long it can chase its target
	Timer int

	// AoE when detonated
	Aoe float64

	// Whether or not hitting a wall will kill it
	DieOnWall bool

	// Whether or not it will explode as designed if it dies without reaching its
	// target
	EffectOnlyOnTarget bool

	Asploded bool
}

func (g *Game) MakeHeatSeeker(pos linear.Vec2, entParams BaseEntParams, hsParams HeatSeekerParams) {
	mine := HeatSeeker{
		BaseEnt: BaseEnt{
			Side_:        10,
			CurrentLevel: GidInvadersStart,
			Position:     pos,
		},
		HeatSeekerParams: hsParams,
	}
	mine.BaseEnt.StatsInst = stats.Make(stats.Base{
		Health: entParams.Health,
		Mass:   entParams.Mass,
		Size:   entParams.Size,
		Acc:    entParams.Acc,
	})
	g.AddEnt(&mine)
}

type massCondition struct {
	Duration int
}

func (mc *massCondition) Supply(mana Mana) Mana {
	return mana
}

func (mc *massCondition) ModifyBase(b stats.Base) stats.Base {
	b.Mass *= 1.5
	base.Log().Printf("Returning mass %v", b.Mass)
	return b
}
func (mc *massCondition) ModifyDamage(damage stats.Damage) stats.Damage {
	return damage
}
func (mc *massCondition) CauseDamage() stats.Damage {
	return stats.Damage{}
}

func (mc *massCondition) Think(game *Game) {
	mc.Duration--
}

func (mc *massCondition) Kill(game *Game) {
	mc.Duration = 0
}

func (mc *massCondition) Phase() Phase {
	if mc.Duration <= 0 {
		return PhaseComplete
	}
	return PhaseRunning
}
func (mc *massCondition) Draw(id Gid, game *Game, side int) {
}

func (hs *HeatSeeker) Dead() bool {
	if hs.Asploded {
		return true
	}
	return hs.BaseEnt.Dead()
}

func (hs *HeatSeeker) Asplode(g *Game) {
	hs.Asploded = true
	for _, ent := range g.Ents {
		if ent == hs {
			continue
		}
		player, ok := ent.(*PlayerEnt)
		if !ok {
			continue
		}
		if hs.Pos().Sub(player.Pos()).Mag2() <= hs.Aoe*hs.Aoe {
			for _, damage := range hs.Damages {
				player.Stats().ApplyDamage(damage)
			}
			for _, conditionMaker := range hs.ConditionMakers {
				condition := effect_makers[conditionMaker.Name](conditionMaker.Params)
				player.Processes[g.NextId()] = condition
			}
		}
	}
}

func (hs *HeatSeeker) Think(g *Game) {
	hs.BaseEnt.Think(g)
	hs.Timer--
	if hs.Timer == 0 {
		hs.Asplode(g)
		return
	}
	targetEnt := g.Ents[hs.TargetGid]
	if targetEnt == nil {
		hs.Asplode(g)
		return
	}
	target, ok := targetEnt.(*PlayerEnt)
	if !ok {
		hs.Asplode(g)
		return
	}
	if target.Pos().Sub(hs.Position).Mag() < target.Stats().Size()+hs.Stats().Size() {
		hs.Asplode(g)
		return
	}
	acc := target.Pos().Sub(hs.Position).Norm().Scale(hs.Stats().MaxAcc())
	hs.ApplyForce(acc)
}

func (m *HeatSeeker) Draw(g *Game) {
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
func (m *HeatSeeker) Supply(mana Mana) Mana { return Mana{} }
func (m *HeatSeeker) Walls() [][]linear.Vec2 {
	return nil
}
