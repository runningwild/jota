package game

import (
	"encoding/gob"
	"github.com/runningwild/cgf"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/linear"
)

type PlayerEnt struct {
	BaseEnt
	Champ int
}

func init() {
	gob.Register(&PlayerEnt{})
}

func (p *PlayerEnt) Think(g *Game) {
	p.BaseEnt.Think(g)
}

func (p *PlayerEnt) Supply(supply Mana) Mana {
	for _, process := range p.Processes {
		supply = process.Supply(supply)
	}
	return supply
}

// AddPlayers adds numPlayers to the specified side.  In standard game mode side
// should be zero, otherwise it should be between 0 and number of side - 1,
// inclusive.
func (g *Game) AddPlayers(gids []Gid, side int) {
	if side < 0 || side >= len(g.Level.Room.Starts) {
		base.Error().Fatalf("Got side %d, but this level only supports sides from 0 to %d.", len(g.Level.Room.Starts)-1)
	}
	for i := range gids {
		var p PlayerEnt
		p.StatsInst = stats.Make(stats.Base{
			Health: 1000,
			Mass:   750,
			Acc:    300.0,
			Turn:   0.07,
			Rate:   0.5,
			Size:   12,
			Vision: 600,
		})

		// Evenly space the players on a circle around the starting position.
		rot := (linear.Vec2{25, 0}).Rotate(float64(i) * 2 * 3.1415926535 / float64(len(gids)))
		p.Position = g.Level.Room.Starts[side].Add(rot)

		p.Side_ = side
		p.Gid = gids[i]
		p.Processes = make(map[int]Process)
		g.AddEnt(&p)
		gids = append(gids, p.Gid)
	}
}

type AiMaker func(name string, engine *cgf.Engine, gid Gid) Ai

var ai_maker AiMaker

func RegisterAiMaker(maker AiMaker) {
	ai_maker = maker
}

type Ai interface {
	Start()
	Stop()
	Terminate()
}
