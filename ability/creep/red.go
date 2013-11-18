package creep

import (
	"encoding/gob"
	"github.com/runningwild/jota/ability"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/linear"
)

func makeAsplode(params map[string]float64) game.Ability {
	var a asplode
	a.id = ability.NextAbilityId()
	a.startRadius = params["startRadius"]
	a.endRadius = params["endRadius"]
	a.durationThinks = int(params["durationThinks"])
	a.dps = params["dps"]
	return &a
}

func init() {
	game.RegisterAbility("asplode", makeAsplode)
	gob.Register(&asplode{})
}

type asplode struct {
	id             int
	startRadius    float64
	endRadius      float64
	durationThinks int
	dps            float64
}

func (a *asplode) Input(ent game.Ent, g *game.Game, pressAmt float64, trigger bool) {
	if trigger && !ent.Dead() {
		// Kill ent and put down explosion
		ent.Suicide()
		g.Processes = append(g.Processes, &asplosionProc{
			StartRadius:    a.startRadius,
			EndRadius:      a.endRadius,
			DurationThinks: a.durationThinks,
			Dps:            a.dps,
			Pos:            ent.Pos(),
		})
	}
}
func (a *asplode) Think(ent game.Ent, game *game.Game) {

}
func (a *asplode) Draw(ent game.Ent, game *game.Game) {

}
func (a *asplode) IsActive() bool {
	return false
}

type asplosionProc struct {
	ability.NullCondition
	DurationThinks int
	NumThinks      int
	StartRadius    float64
	EndRadius      float64
	CurrentRadius  float64
	Dps            float64
	Pos            linear.Vec2
	Killed         bool
}

func (p *asplosionProc) Supply(mana game.Mana) game.Mana {
	return game.Mana{}
}
func (p *asplosionProc) Think(g *game.Game) {
	p.NumThinks++
	p.CurrentRadius = float64(p.NumThinks)/float64(p.DurationThinks)*(p.EndRadius-p.StartRadius) + p.StartRadius
	for _, ent := range g.Ents {
		if ent.Pos().Sub(p.Pos).Mag2() <= p.CurrentRadius*p.CurrentRadius {
			ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, p.Dps})
		}
	}
}
func (p *asplosionProc) Kill(g *game.Game) {
	p.Killed = true
}
func (p *asplosionProc) Dead() bool {
	return p.Killed || p.NumThinks > p.DurationThinks
}
