package control_point

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/jota/ability"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/jota/texture"
	"github.com/runningwild/linear"
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
	proc, ok := cp.Processes[sc.id].(*omniDrain)
	if !ok {
		base.Log().Printf("Started draining for %v", ent.Id())
		cp.Processes[sc.id] = &omniDrain{Gid: cp.Gid}
		return
	}
	if trigger {
		base.Log().Printf("Stored %2.3v", proc.Stored)
		g.AddEnt(ent)
		delete(cp.Processes, sc.id)
		addCreepsToSide(g, cp.Pos(), 1, cp.Side())
	}
}
func (sc *spawnCreeps) Think(ent game.Ent, game *game.Game) {

}
func (sc *spawnCreeps) Draw(ent game.Ent, game *game.Game) {

}
func (sc *spawnCreeps) IsActive() bool {
	return false
}

type CreepEnt struct {
	game.BaseEnt
}

func init() {
	gob.Register(&CreepEnt{})
}
func (c *CreepEnt) Think(g *game.Game) {
	c.BaseEnt.Think(g)
}

func (c *CreepEnt) Supply(mana game.Mana) game.Mana {
	return mana
}
func (c *CreepEnt) Draw(g *game.Game) {
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.01)
	base.SetUniformF("status_bar", "outer", 0.03)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(255, 255, 255, 255)
	texture.Render(c.Position.X-100, c.Position.Y-100, 200, 200)
	base.SetUniformF("status_bar", "inner", 0.04)
	base.SetUniformF("status_bar", "outer", 0.045)
	base.SetUniformF("status_bar", "buffer", 0.01)
	base.SetUniformF("status_bar", "frac", 1.0)
	texture.Render(c.Position.X-100, c.Position.Y-100, 200, 200)
	base.EnableShader("")
}

func addCreepsToSide(g *game.Game, pos linear.Vec2, count, side int) {
	if side < 0 || side >= len(g.Level.Room.SideData) {
		base.Error().Fatalf("Got side %d, but this level only supports sides from 0 to %d.", len(g.Level.Room.SideData)-1)
	}
	for i := 0; i < count; i++ {
		var c CreepEnt
		c.StatsInst = stats.Make(stats.Base{
			Health: 1000,
			Mass:   750,
			Acc:    300.0,
			Rate:   0.0,
			Size:   8,
			Vision: 600,
		})

		// Evenly space the players on a circle around the starting position.
		rot := (linear.Vec2{50, 0}).Rotate(float64(i) * 2 * 3.1415926535 / float64(count))
		c.Position = pos.Add(rot)

		c.Side_ = side
		c.Gid = g.NextGid()

		// for _, ability := range g.Champs[playerData.champ].Abilities {
		// 	c.Abilities_ = append(
		// 		c.Abilities_,
		// 		ability_makers[ability.Name](ability.Params))
		// }

		// if playerData.gid[0:2] == "Ai" {
		// 	c.BindAi("simple", g.local.Engine)
		// }

		g.AddEnt(&c)
	}
}
