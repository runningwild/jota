package game

import (
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/los"
	"github.com/runningwild/magnus/stats"
)

// Moba base ent
type FrozenThrone struct {
	BaseEnt
	Los *los.Los
}

func (g *Game) MakeFrozenThrones() {
	for i, data := range g.Levels[GidInvadersStart].Room.Moba.SideData {
		ft := FrozenThrone{
			BaseEnt: BaseEnt{
				Side_:        i,
				CurrentLevel: GidInvadersStart,
				Position:     data.Base,
			},
			Los: los.Make(LosMaxDist),
		}
		ft.BaseEnt.StatsInst = stats.Make(100000, 1000000, 0, 0, 1)
		g.AddEnt(&ft)
	}
}

func (ft *FrozenThrone) Draw(g *Game, ally bool) {}
func (ft *FrozenThrone) Supply(mana Mana) Mana   { return Mana{} }
func (ft *FrozenThrone) Walls() [][]linear.Vec2 {
	return [][]linear.Vec2{
		[]linear.Vec2{
			ft.Position.Add(linear.Vec2{-50, -50}),
			ft.Position.Add(linear.Vec2{-50, 50}),
			ft.Position.Add(linear.Vec2{50, 50}),
			ft.Position.Add(linear.Vec2{50, -50}),
		},
	}
}
