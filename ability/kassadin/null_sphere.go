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
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
)

func makeNullSphere(params map[string]int) game.Ability {
	var ns nullSphere
	ns.id = ability.NextAbilityId()
	ns.cost = float64(params["cost"])
	return &ns
}

func init() {
	game.RegisterAbility("nullSphere", makeNullSphere)
}

type nullSphere struct {
	ability.NonRendering
	id   int
	fire int
	cost float64
}

func (ns *nullSphere) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	var ret []cgf.Event
	if keyPress {
		ret = append(ret, addNullSphereCastProcessEvent{
			PlayerGid: gid,
			ProcessId: ns.id,
			Cost:      ns.cost,
		})
	} else {
		ret = append(ret, removeNullSphereCastProcessEvent{
			PlayerGid: gid,
			ProcessId: ns.id,
		})
	}
	return ret, keyPress
}

func (ns *nullSphere) Deactivate(gid game.Gid) []cgf.Event {
	ns.fire = 0
	return nil
}

func (ns *nullSphere) Respond(gid game.Gid, group gin.EventGroup) bool {
	if found, event := group.FindEvent(gin.AnySpace); found && event.Type == gin.Press {
		ns.fire++
		return true
	}
	return false
}

func (ns *nullSphere) Think(gid game.Gid, g *game.Game, mouse linear.Vec2) ([]cgf.Event, bool) {
	var ret []cgf.Event
	for ; ns.fire > 0; ns.fire-- {
		ret = append(ret, addNullSphereFireEvent{
			PlayerGid: gid,
			ProcessId: ns.id,
		})
	}
	return ret, false
}

type nullSphereCastProcess struct {
	ability.BasicPhases
	ability.NullCondition
	PlayerGid game.Gid
	Stored    game.Mana
	Cost      float64

	targetGid game.Gid
}

func (p *nullSphereCastProcess) Supply(supply game.Mana) game.Mana {
	p.Stored[game.ColorBlue] += supply[game.ColorBlue]
	supply[game.ColorBlue] = 0
	return supply
}

func (p *nullSphereCastProcess) Think(g *game.Game) {
	p.Stored[game.ColorBlue] *= 0.98
	p.targetGid = ""
	ent := g.Ents[p.PlayerGid]
	if ent == nil {
		return
	}
	player, ok := ent.(*game.Player)
	if !ok {
		return
	}
	target := nullSphereTarget(g, player)
	if target == nil {
		return
	}
	p.targetGid = target.Id()
}

// TODO: This function really needs to take not just the side, but the player
// that this is being drawn for.
func (p *nullSphereCastProcess) Draw(gid game.Gid, g *game.Game, side int) {
	player, _ := g.Ents[p.PlayerGid].(*game.Player)
	target, _ := g.Ents[p.targetGid].(*game.Player)
	if player == nil {
		return
	}
	if side != player.Side() {
		return
	}
	if target != nil {
		base.EnableShader("circle")
		base.SetUniformF("circle", "edge", 0.99)
		gl.Color4ub(200, 200, 10, 150)
		size := target.Stats().Size() + 25
		texture.Render(
			target.Pos().X-size,
			target.Pos().Y-size,
			2*size,
			2*size)
		base.EnableShader("")
	}
	ready := int(p.Stored[game.ColorBlue] / p.Cost)
	base.EnableShader("status_bar")
	if ready == 0 {
		gl.Color4ub(255, 0, 0, 255)
	} else {
		gl.Color4ub(0, 255, 0, 255)
	}
	var outer float32 = 0.2
	var increase float32 = 0.01
	frac := p.Stored[game.ColorBlue] / p.Cost
	base.SetUniformF("status_bar", "frac", float32(frac-float64(ready)))
	base.SetUniformF("status_bar", "inner", outer-increase*float32(ready+1))
	base.SetUniformF("status_bar", "outer", outer)
	base.SetUniformF("status_bar", "buffer", 0.01)
	texture.Render(player.Pos().X-100, player.Pos().Y-100, 200, 200)
	if ready > 0 {
		base.SetUniformF("status_bar", "frac", 1.0)
		base.SetUniformF("status_bar", "inner", outer-float32(ready)*increase)
		base.SetUniformF("status_bar", "outer", outer)
		texture.Render(player.Pos().X-100, player.Pos().Y-100, 200, 200)
	}
	base.EnableShader("")
}

type addNullSphereCastProcessEvent struct {
	PlayerGid game.Gid
	ProcessId int
	Cost      float64
}

func init() {
	gob.Register(addNullSphereCastProcessEvent{})
}

func (e addNullSphereCastProcessEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	player.Processes[100+e.ProcessId] = &nullSphereCastProcess{
		PlayerGid: e.PlayerGid,
		Cost:      e.Cost,
	}
}

type removeNullSphereCastProcessEvent struct {
	PlayerGid game.Gid
	ProcessId int
	Cost      float64
}

func init() {
	gob.Register(removeNullSphereCastProcessEvent{})
}

func (e removeNullSphereCastProcessEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	proc := player.Processes[100+e.ProcessId]
	if proc != nil {
		proc.Kill(g)
	}
}

type addNullSphereFireEvent struct {
	PlayerGid game.Gid
	ProcessId int
}

func init() {
	gob.Register(addNullSphereFireEvent{})
}

func nullSphereTarget(g *game.Game, player *game.Player) game.Ent {
	var bestEnt game.Ent
	var bestDistSq float64 = player.Stats().Vision() * player.Stats().Vision()
	g.DoForEnts(func(gid game.Gid, ent game.Ent) {
		if ent.Side() == player.Side() {
			// Don't target anything on the same side
			return
		}
		if _, ok := ent.(*game.Player); !ok {
			// Only target players
			return
		}
		if !g.ExistsLos(player.Pos(), ent.Pos()) {
			// Only target players that we have los to
			return
		}
		distSq := player.Pos().Sub(ent.Pos()).Mag2()
		if distSq <= bestDistSq {
			bestDistSq = distSq
			bestEnt = ent
		}
	})
	return bestEnt
}

func (e addNullSphereFireEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	proc := player.Processes[100+e.ProcessId]
	if proc == nil {
		return
	}
	nsProc, ok := proc.(*nullSphereCastProcess)
	if !ok {
		return
	}
	target := nullSphereTarget(g, player)
	if target == nil {
		return
	}
	if nsProc.Stored[game.ColorBlue] < nsProc.Cost {
		// Can't cast until you've stored up the minimum amount
		return
	}
	nsProc.Stored[game.ColorBlue] -= nsProc.Cost
	size := player.Stats().Size()
	g.MakeHeatSeeker(
		player.Pos().Add(target.Pos().Sub(player.Pos()).Norm().Scale(player.Stats().Size()+size)),
		game.BaseEntParams{
			Health: 100,
			Mass:   100,
			Size:   size,
			Acc:    nsProc.Cost / 2,
		},
		game.HeatSeekerParams{
			Target:             target.Id(),
			Damages:            []stats.Damage{{stats.DamageFire, 50}},
			ConditionMakers:    []game.ConditionMaker{{"silence", map[string]int{"duration": 300}}},
			Timer:              300,
			Aoe:                50,
			DieOnWall:          false,
			EffectOnlyOnTarget: true,
		})
}
