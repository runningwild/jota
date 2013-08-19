package game

import (
	"bytes"
	"encoding/json"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/los"
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
		err := json.NewDecoder(bytes.NewBuffer([]byte(`
        {
          "Base": {
            "Max_turn": 0.0,
            "Max_acc": 0.0,
            "Mass": 1000000,
            "Max_rate": 1000,
            "Influence": 1000,
            "Health": 100000
          },
          "Dynamic": {
            "Health": 100000
          }
        }
      `))).Decode(&ft.BaseEnt.StatsInst)
		if err != nil {
			panic(err)
		}
		g.AddEnt(&ft)
	}
}

func (ft *FrozenThrone) Draw(g *Game)          {}
func (ft *FrozenThrone) Supply(mana Mana) Mana { return Mana{} }
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
