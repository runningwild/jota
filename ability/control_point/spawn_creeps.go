package control_point

import (
	"encoding/gob"
	"github.com/runningwild/jota/ability"
	"github.com/runningwild/jota/game"
)

func makeSpawnCreeps(params map[string]int) game.Ability {
	var sc spawnCreeps
	sc.id = ability.NextAbilityId()
	return &sc
}

func init() {
	game.RegisterAbility("spawnCreeps", makeSpawnCreeps)
	gob.Register(&spawnCreeps{})
}

type spawnCreeps struct {
	id      int
	health  float64
	damage  float64
	trigger float64
	mass    float64
	cost    float64
	fire    int
}

// Typical process for draining all mana possible.
type omniDrain struct {
	ability.NullCondition

	// Gid of the Ent with this Process
	Gid game.Gid

	// The number of multiples of Unit currently stored
	Stored game.Mana

	Killed bool
}

func (p *omniDrain) Draw(src, obs game.Gid, game *game.Game) {
}
func (p *omniDrain) Supply(mana game.Mana) game.Mana {
	for color := range mana {
		p.Stored[color] += mana[color]
	}
	return game.Mana{}
}
func (p *omniDrain) Think(g *game.Game) {
	if _, ok := g.Ents[p.Gid]; ok {
		// TODO: Get this from the ent's stats once it's a stat.
		retention := 0.98
		for color := range p.Stored {
			p.Stored[color] *= retention
		}
	} else {
		p.Killed = true
	}
}
func (p *omniDrain) Kill(g *game.Game) {
	p.Killed = true
}
func (p *omniDrain) Dead() bool {
	return p.Killed
}

func (sc *spawnCreeps) Input(ent game.Ent, g *game.Game, pressAmt float64, trigger bool) {
	cp := ent.(*game.ControlPoint)
	if pressAmt == 0 {
		delete(cp.Processes, sc.id)
		return
	}
	_, ok := cp.Processes[sc.id].(*omniDrain)
	if !ok {
		cp.Processes[sc.id] = &omniDrain{Gid: cp.Gid}
		return
	}
	if trigger {
		g.AddEnt(ent)
		delete(cp.Processes, sc.id)
		g.AddCreeps(cp.Pos(), 1, cp.Side(), map[string]interface{}{"target": cp.Targets[0]})
	}
}
func (sc *spawnCreeps) Think(ent game.Ent, game *game.Game) {

}
func (sc *spawnCreeps) Draw(ent game.Ent, game *game.Game) {

}
func (sc *spawnCreeps) IsActive() bool {
	return false
}
