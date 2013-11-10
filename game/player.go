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

func (p *PlayerEnt) Type() EntType {
	return EntTypePlayer
}

func init() {
	gob.Register(&PlayerEnt{})
}

func (p *PlayerEnt) Think(g *Game) {
	p.BaseEnt.Think(g)
}

func (p *PlayerEnt) Supply(mana Mana) Mana {
	base.DoOrdered(p.Processes, func(a, b int) bool { return a < b }, func(id int, proc Process) {
		mana = proc.Supply(mana)
	})
	return mana
}

// AddPlayers adds numPlayers to the specified side.  In standard game mode side
// should be zero, otherwise it should be between 0 and number of side - 1,
// inclusive.
func (g *Game) AddPlayers(players []*PlayerData) {
	bySide := make(map[int][]addPlayerData)
	for _, player := range players {
		bySide[player.Side] = append(bySide[player.Side], addPlayerData{player.PlayerGid, player.ChampIndex})
	}
	for side, players := range bySide {
		g.addPlayersToSide(players, side)
	}
}

type addPlayerData struct {
	gid   Gid
	champ int
}

func (g *Game) addPlayersToSide(playerDatas []addPlayerData, side int) {
	if side < 0 || side >= len(g.Level.Room.SideData) {
		base.Error().Fatalf("Got side %d, but this level only supports sides from 0 to %d.", len(g.Level.Room.SideData)-1)
	}
	for i, playerData := range playerDatas {
		var p PlayerEnt
		p.StatsInst = stats.Make(stats.Base{
			Health: 1000,
			Mass:   750,
			Acc:    300.0,
			Turn:   0.07,
			Rate:   0.5,
			Size:   12,
			Vision: 500,
		})

		// Evenly space the players on a circle around the starting position.
		rot := (linear.Vec2{25, 0}).Rotate(float64(i) * 2 * 3.1415926535 / float64(len(playerDatas)))
		p.Position = g.Level.Room.SideData[side].Base.Add(rot)

		p.Side_ = side
		p.Gid = playerData.gid
		p.Processes = make(map[int]Process)

		for _, ability := range g.Champs[playerData.champ].Abilities {
			p.Abilities_ = append(
				p.Abilities_,
				ability_makers[ability.Name](ability.Params))
		}

		if playerData.gid[0:2] == "Ai" {
			// p.BindAi("simple", g.local.Engine)
		}

		g.AddEnt(&p)
	}
}

type AiMaker func(name string, engine *cgf.Engine, gid Gid) Ai

var ai_maker AiMaker

func RegisterAiMaker(maker AiMaker) {
	ai_maker = maker
}

type Ai interface {
	SetParam(name string, value interface{})
	Start()
	Stop()
	Terminate()
}
