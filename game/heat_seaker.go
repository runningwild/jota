package game

import (
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/linear"
)

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
	Params map[string]float64
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
			Side_:    10,
			Position: pos,
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

func (mc *massCondition) Dead() bool {
	return mc.Duration == 0
}
func (mc *massCondition) Draw(id Gid, game *Game, side int) {
}

func (hs *HeatSeeker) Type() EntType {
	return EntTypeProjectile
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

func (m *HeatSeeker) Supply(mana Mana) Mana { return Mana{} }
