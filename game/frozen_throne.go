package game

import (
	"github.com/runningwild/linear"
	"github.com/runningwild/jota/los"
	"github.com/runningwild/jota/stats"
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
		ft.BaseEnt.StatsInst = stats.Make(stats.Base{
			Health: 100000,
			Mass:   1000000,
			Rate:   1,
			Size:   100,
			Vision: 900,
		})
		g.AddEnt(&ft)
	}
}

func (ft *FrozenThrone) Draw(g *Game, side int) {}
func (ft *FrozenThrone) Supply(mana Mana) Mana  { return Mana{} }
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
