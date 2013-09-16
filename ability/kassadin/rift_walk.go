package kassadin

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/ability"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/magnus/stats"
	"math"
)

func makeRiftWalk(params map[string]int) game.Ability {
	var b riftWalk
	b.id = ability.NextAbilityId()
	b.force = float64(params["force"])
	return &b
}

type riftWalk struct {
	id      int
	trigger bool
	force   float64
}

func init() {
	game.RegisterAbility("riftWalk", makeRiftWalk)
}

func (rw *riftWalk) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	var ret []cgf.Event
	if keyPress {
		ret = append(ret, addRiftWalkEvent{
			PlayerGid: gid,
			ProcessId: rw.id,
			Force:     rw.force,
		})
	} else {
		ret = append(ret, removeRiftWalkEvent{
			PlayerGid: gid,
			ProcessId: rw.id,
		})
	}
	return ret, keyPress
}

func (rw *riftWalk) Deactivate(gid game.Gid) []cgf.Event {
	rw.trigger = false
	return nil
}

func (rw *riftWalk) Respond(gid game.Gid, group gin.EventGroup) bool {
	if found, event := group.FindEvent(gin.AnySpace); found && event.Type == gin.Press {
		rw.trigger = true
		return true
	}
	return false
}

func (rw *riftWalk) Think(gid game.Gid, g *game.Game, mouse linear.Vec2) ([]cgf.Event, bool) {
	var ret []cgf.Event
	if rw.trigger {
		rw.trigger = false
		ret = append(ret, addRiftWalkFireEvent{
			PlayerGid: gid,
			ProcessId: rw.id,
		})
	}
	return ret, false
}

func (rw *riftWalk) Draw(gid game.Gid, g *game.Game, side int) {}

type addRiftWalkEvent struct {
	PlayerGid game.Gid
	ProcessId int
	Force     float64
}

func init() {
	gob.Register(addRiftWalkEvent{})
}

func (e addRiftWalkEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	player.Processes[100+e.ProcessId] = &riftWalkProcess{
		BasicPhases: ability.BasicPhases{game.PhaseRunning},
		PlayerGid:   e.PlayerGid,
		Id:          e.ProcessId,
		Force:       e.Force,
	}
}

type removeRiftWalkEvent struct {
	PlayerGid game.Gid
	ProcessId int
}

func init() {
	gob.Register(removeRiftWalkEvent{})
}

func (e removeRiftWalkEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	proc := player.Processes[100+e.ProcessId]
	if proc != nil {
		proc.Kill(g)
	}
	// delete(player.Processes, 100+e.ProcessId)
	// }
}

func init() {
	gob.Register(&riftWalkProcess{})
}

type riftWalkProcess struct {
	ability.BasicPhases
	ability.NullCondition
	PlayerGid game.Gid
	Id        int
	Force     float64
	Stored    game.Mana
}

func (p *riftWalkProcess) Supply(supply game.Mana) game.Mana {
	for _, color := range []game.Color{game.ColorBlue, game.ColorGreen} {
		p.Stored[color] *= 0.98
		p.Stored[color] += supply[color]
		supply[color] = 0
	}
	return supply
}

func (p *riftWalkProcess) Think(g *game.Game) {
	for i := range p.Stored {
		p.Stored[i] *= 0.98
	}
	// ent := g.Ents[p.PlayerGid]
	// player, ok := ent.(*game.Player)
	// if !ok {
	// 	return
	// }
}

func (p *riftWalkProcess) GetVals() (distance, radius float64) {
	distance = math.Sqrt(p.Stored[game.ColorGreen]) * 10
	radius = math.Sqrt(p.Stored[game.ColorBlue]) / p.Force * 50000
	return
}

func (p *riftWalkProcess) Draw(gid game.Gid, g *game.Game, side int) {
	player, ok := g.Ents[p.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	if side != player.Side() {
		return
	}
	dist, radius := p.GetVals()
	dest := player.Pos().Add((linear.Vec2{dist, 0}).Rotate(player.Angle))
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4d(1, 1, 1, 1)
	gl.Begin(gl.LINES)
	gl.Vertex2d(gl.Double(player.Pos().X), gl.Double(player.Pos().Y))
	gl.Vertex2d(gl.Double(dest.X), gl.Double(dest.Y))
	gl.End()
	n := 20
	gl.Begin(gl.LINES)
	for i := 0; i < n; i++ {
		v1 := dest.Add((linear.Vec2{radius, 0}).Rotate(float64(i) / float64(n) * 2 * math.Pi))
		v2 := dest.Add((linear.Vec2{radius, 0}).Rotate(float64(i+1) / float64(n) * 2 * math.Pi))
		gl.Vertex2d(gl.Double(v1.X), gl.Double(v1.Y))
		gl.Vertex2d(gl.Double(v2.X), gl.Double(v2.Y))
	}
	gl.End()
}

type addRiftWalkFireEvent struct {
	PlayerGid game.Gid
	ProcessId int
}

func init() {
	gob.Register(addRiftWalkFireEvent{})
}
func (e addRiftWalkFireEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	proc := player.Processes[100+e.ProcessId]
	if proc == nil {
		return
	}
	rwProc, ok := proc.(*riftWalkProcess)
	if !ok {
		return
	}
	dist, radius := rwProc.GetVals()
	rwProc.Stored = game.Mana{}
	dest := player.Pos().Add((linear.Vec2{dist, 0}).Rotate(player.Angle))
	for _, ent := range g.Ents {
		if ent == player {
			continue
		}
		doDamage := false
		if ent.Pos().Sub(dest).Mag() <= radius+ent.Stats().Size() {
			angle := dest.Sub(ent.Pos()).Angle()
			ent.ApplyForce((linear.Vec2{-rwProc.Force, 0}).Rotate(angle))
			doDamage = true
		} else {
			ray := dest.Sub(player.Pos())
			perp := ray.Cross().Norm()
			scaledPerp := perp.Scale(ent.Stats().Size() + player.Stats().Size())
			sideRay0 := linear.Seg2{player.Pos().Add(scaledPerp), dest.Add(scaledPerp)}
			sideRay1 := linear.Seg2{player.Pos().Sub(scaledPerp), dest.Sub(scaledPerp)}
			if sideRay0.Left(ent.Pos()) != sideRay1.Left(ent.Pos()) {
				// We know the ent lies between sideRays 0 and 1, now we need to make
				// sure it lies between src and dst.
				forward := ent.Pos().Sub(dest)
				backward := ent.Pos().Sub(player.Pos())
				if (forward.Dot(ray) < 0) != (backward.Dot(ray) < 0) {
					if (linear.Seg2{player.Pos(), dest}).Left(ent.Pos()) {
						ent.ApplyForce(perp.Scale(rwProc.Force))
					} else {
						ent.ApplyForce(perp.Scale(-rwProc.Force))
					}
					doDamage = true
				}
			}
		}
		if doDamage {
			ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, 50})
		}
	}
	player.SetPos(dest)
}
