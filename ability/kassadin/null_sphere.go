package kassadin

import (
	"encoding/gob"
	// gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/ability"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
	// "github.com/runningwild/magnus/texture"
	// "math"
)

func makeNullSphere(params map[string]int) game.Ability {
	var ns nullSphere
	ns.id = ability.NextAbilityId()
	return &ns
}

func init() {
	game.RegisterAbility("nullSphere", makeNullSphere)
}

type nullSphere struct {
	id   int
	fire int
}

func (ns *nullSphere) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	var ret []cgf.Event
	if keyPress {
		ret = append(ret, addNullSphereCastProcessEvent{
			PlayerGid: gid,
			ProcessId: ns.id,
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
	base.Log().Printf("NS Deactivate")
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

func (ns *nullSphere) Draw(gid game.Gid, g *game.Game, side int) {
	// Draw something showing how much mana has been stored
}

type nullSphereCastProcess struct {
	ability.BasicPhases
	ability.NullCondition
	PlayerGid game.Gid
	Stored    game.Mana
}

func (p *nullSphereCastProcess) Supply(supply game.Mana) game.Mana {
	p.Stored[game.ColorBlue] *= 0.98
	p.Stored[game.ColorBlue] += supply[game.ColorBlue]
	supply[game.ColorBlue] = 0
	return supply
}

func (p *nullSphereCastProcess) Think(g *game.Game) {
	// if p.Remaining.Magnitude() == 0 {
	// 	size := 10.0
	// 	g.MakeHeatSeeker(
	// 		player.Pos().Add(target.Pos().Sub(player.Pos()).Norm().Scale(player.Stats().Size()+size)),
	// 		game.BaseEntParams{
	// 			Health: 100,
	// 			Mass:   100,
	// 			Size:   size,
	// 			Acc:    40,
	// 		},
	// 		game.HeatSeekerParams{
	// 			Target:             p.TargetGid,
	// 			Damage:             10,
	// 			Timer:              300,
	// 			Aoe:                50,
	// 			DieOnWall:          false,
	// 			EffectOnlyOnTarget: true,
	// 		})
	// 	p.The_phase = game.PhaseComplete
	// }
}

func (p *nullSphereCastProcess) Draw(gid game.Gid, g *game.Game, side int) {
}

type addNullSphereCastProcessEvent struct {
	PlayerGid game.Gid
	ProcessId int
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
	}
}

type removeNullSphereCastProcessEvent struct {
	PlayerGid game.Gid
	ProcessId int
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
	delete(player.Processes, 100+e.ProcessId)
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
	var bestDistSq float64 = 1e9
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
		if distSq < bestDistSq {
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
	amt := 100.0
	if nsProc.Stored[game.ColorBlue] < amt {
		// Can't cast until you've stored up the minimum amount
		// TODO: Make the minimum value a parameter on the ability or something
		return
	}
	nsProc.Stored[game.ColorBlue] -= amt
	size := player.Stats().Size()
	g.MakeHeatSeeker(
		player.Pos().Add(target.Pos().Sub(player.Pos()).Norm().Scale(player.Stats().Size()+size)),
		game.BaseEntParams{
			Health: 100,
			Mass:   100,
			Size:   size,
			Acc:    amt / 2,
		},
		game.HeatSeekerParams{
			Target:             target.Id(),
			Damage:             10,
			Timer:              300,
			Aoe:                50,
			DieOnWall:          false,
			EffectOnlyOnTarget: true,
		})
}
