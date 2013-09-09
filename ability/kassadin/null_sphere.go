package kassadin

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/ability"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/magnus/texture"
	// "math"
)

func makeNullSphere(params map[string]int) game.Ability {
	var ns nullSphere
	ns.id = ability.NextAbilityId()

	// ns.force = float64(params["force"])
	// ns.angle = float64(params["angle"]) / 180 * 3.14159
	return &ns
}

func init() {
	game.RegisterAbility("nullSphere", makeNullSphere)
}

type nullSphere struct {
	id int

	force float64
	angle float64

	// number of times the player has hit tab.
	tab int

	// number of times the player has hit fire.
	fire int

	// Id of the current target
	targetId game.Gid

	// Id of the player using this ability
	sourceId game.Gid
}

func (ns *nullSphere) nextTarget(g *game.Game) {
	sourceEnt, ok := g.Ents[ns.sourceId]
	if !ok {
		return
	}
	side := sourceEnt.Side()

	// Because we have to iterate through ents via this closure, pos will keep
	// track of some logic, it has the following meanings:
	// pos == 0: haven't found the previously selected entity yet.
	// pos == 1: just found the previously selected entity *or* there was no
	//					 previously selected entity.
	// pos == 2: the next entity to select has been chosen
	// TODO: Implement a tabbing system that does something aesthetically pleasing
	// like tab in order of range or angle or something.
	pos := 0
	if ns.targetId == "" {
		pos = 1
	}
	var firstId game.Gid
	base.Log().Printf("Starting pos %d, id %v\n", pos, ns.targetId)
	g.DoForEnts(func(gid game.Gid, ent game.Ent) {
		if pos == 2 {
			return
		}
		if ent.Side() == side {
			return
		}
		target, ok := ent.(*game.Player)
		if !ok {
			return
		}
		if sourceEnt.Pos().Sub(target.Pos()).Mag() > sourceEnt.Stats().Vision() {
			return
		}
		if !g.ExistsLos(sourceEnt.Pos(), target.Pos()) {
			return
		}
		if firstId == "" {
			firstId = gid
		}
		base.Log().Printf("Pos %d, checking %v.", pos, gid)
		if pos == 0 && gid == ns.targetId {
			pos = 1
		} else if pos == 1 {
			pos = 2
			ns.targetId = gid
		}
	})
	if pos < 2 {
		ns.targetId = firstId
	}
}

func (ns *nullSphere) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	if keyPress == false {
		return nil, true
	}
	base.Log().Printf("NS Activate")
	ns.sourceId = gid
	ns.tab = 1
	return nil, true
}

func (ns *nullSphere) Deactivate(gid game.Gid) []cgf.Event {
	base.Log().Printf("NS Deactivate")
	ns.targetId = ""
	return nil
}

func (ns *nullSphere) Respond(gid game.Gid, group gin.EventGroup) bool {
	if found, event := group.FindEvent(gin.AnyTab); found {
		if event.Type == gin.Press {
			base.Log().Printf("NS Tab")
			ns.tab++
		}
		return true
	}
	if found, event := group.FindEvent(gin.AnySpace); found && event.Type == gin.Press {
		ns.fire++
		return true
	}
	return false
}

func (ns *nullSphere) Think(gid game.Gid, g *game.Game, mouse linear.Vec2) ([]cgf.Event, bool) {
	var ret []cgf.Event
	for ; ns.tab > 0; ns.tab-- {
		base.Log().Printf("NS NextTarget")
		ns.nextTarget(g)
		base.Log().Printf("NS Target: %v", ns.targetId)
	}
	for ; ns.fire > 0; ns.fire-- {
		if ns.targetId != "" {
			ret = append(ret, &nullSphereEvent{ns.id, ns.sourceId, ns.targetId})
		}
	}
	if ns.targetId == "" {
		return nil, true
	}
	return ret, false
}

func (ns *nullSphere) Draw(gid game.Gid, g *game.Game, side int) {
	targetEnt, ok := g.Ents[ns.targetId]
	if !ok {
		return
	}
	pos := targetEnt.Pos()

	base.EnableShader("circle")
	base.SetUniformF("circle", "edge", 0.9)

	// For people on the controlling side this will draw a circle around the area
	// that is being targeted by the control point.
	gl.Color4ub(200, 200, 100, 200)
	texture.Render(
		pos.X-50,
		pos.Y-50,
		2*50,
		2*50)

	base.EnableShader("")
}

type nullSphereEvent struct {
	Id        int
	PlayerGid game.Gid
	TargetGid game.Gid
}

func init() {
	gob.Register(nullSphereEvent{})
}

func (e nullSphereEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	_, ok = g.Ents[e.TargetGid].(*game.Player)
	if !ok {
		return
	}
	player.Processes[100+e.Id] = &nullSphereProcess{
		PlayerGid: e.PlayerGid,
		TargetGid: e.TargetGid,
		Remaining: game.Mana{0, 0, 1000},
	}
	// PUT IT HERE
}

// type removeNullSphereEvent struct {
// 	PlayerGid game.Gid
// 	Id        int
// }

// func init() {
// 	gob.Register(removeNullSphereEvent{})
// }

// func (e removeNullSphereEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	player, ok := g.Ents[e.PlayerGid].(*game.Player)
// 	if !ok {
// 		return
// 	}
// 	proc := player.Processes[100+e.Id]
// 	if proc != nil {
// 		proc.Kill(g)
// 		delete(player.Processes, 100+e.Id)
// 	}
// }

// func init() {
// 	gob.Register(&nullSphereProcess{})
// }

type nullSphereProcess struct {
	ability.BasicPhases
	ability.NullCondition
	PlayerGid game.Gid
	TargetGid game.Gid
	Remaining game.Mana
}

func (p *nullSphereProcess) Supply(supply game.Mana) game.Mana {
	for _, color := range game.AllColors {
		if supply[color] > p.Remaining[color] {
			supply[color] -= p.Remaining[color]
			p.Remaining[color] = 0
		} else {
			p.Remaining[color] -= supply[color]
			supply[color] = 0
		}
	}
	return supply
}

func (p *nullSphereProcess) getPlayers(g *game.Game) (player, target *game.Player, ok bool) {
	playerEnt := g.Ents[p.PlayerGid]
	if playerEnt == nil {
		return nil, nil, false
	}
	targetEnt := g.Ents[p.TargetGid]
	if targetEnt == nil {
		return nil, nil, false
	}
	player, ok = playerEnt.(*game.Player)
	if !ok {
		return nil, nil, false
	}
	target, ok = targetEnt.(*game.Player)
	if !ok {
		return nil, nil, false
	}
	return player, target, true
}

func (p *nullSphereProcess) Think(g *game.Game) {
	player, target, ok := p.getPlayers(g)
	if !ok {
		p.The_phase = game.PhaseComplete
		return
	}
	if p.Remaining.Magnitude() == 0 {
		g.MakeHeatSeeker(
			player.Pos().Add(target.Pos().Sub(player.Pos()).Norm().Scale(10)),
			game.BaseEntParams{
				Health: 100,
				Mass:   100,
				Size:   10,
				Acc:    40,
			},
			game.HeatSeekerParams{
				Target:             p.TargetGid,
				Damage:             10,
				Timer:              300,
				Aoe:                50,
				DieOnWall:          false,
				EffectOnlyOnTarget: true,
			})
		p.The_phase = game.PhaseComplete
	}
}

func (p *nullSphereProcess) Draw(gid game.Gid, g *game.Game, side int) {
}
