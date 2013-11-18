package game

import (
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/linear"
)

type Mine struct {
	BaseEnt
	Damage  float64
	Trigger float64
}

func (g *Game) MakeMine(pos, vel linear.Vec2, health, mass, damage, trigger float64) {
	mine := Mine{
		BaseEnt: BaseEnt{
			Side_:    10,
			Position: pos,
			Velocity: vel,
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

func (m *Mine) Type() EntType {
	return EntTypeObstacle
}
func (m *Mine) Think(g *Game) {
	m.BaseEnt.Think(g)
	prox := 50.0
	for _, ent := range g.local.temp.AllEnts {
		if ent == m {
			continue
		}
		if ent.Pos().Sub(m.Position).Mag2() < prox*prox {
			m.Trigger -= ent.Vel().Sub(m.Velocity).Mag2()
		}
	}
	if m.Trigger <= 0 {
		for _, ent := range g.local.temp.AllEnts {
			if ent.Pos().Sub(m.Position).Mag() < prox {
				ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, m.Damage})
			}
		}
	}
}

func (m *Mine) Supply(mana Mana) Mana { return mana }
