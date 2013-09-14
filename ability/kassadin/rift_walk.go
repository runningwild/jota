package kassadin

// import (
// 	"encoding/gob"
// 	gl "github.com/chsc/gogl/gl21"
// 	"github.com/runningwild/cgf"
// 	"github.com/runningwild/linear"
// 	"github.com/runningwild/magnus/game"
// 	"math"
// )

// func makePull(params map[string]int) game.Ability {
// 	var b pull
// 	b.id = NextAbilityId()
// 	b.force = float64(params["force"])
// 	b.angle = float64(params["angle"]) / 180 * 3.14159
// 	return &b
// }

// func init() {
// 	game.RegisterAbility("pull", makePull)
// }

// type pull struct {
// 	NonResponder
// 	NonThinker
// 	NonRendering

// 	id    int
// 	force float64
// 	angle float64
// }

// func (p *pull) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
// 	ret := []cgf.Event{
// 		addPullEvent{
// 			PlayerGid: gid,
// 			Id:        p.id,
// 			Force:     p.force,
// 			Angle:     p.angle,
// 			Press:     keyPress,
// 		},
// 	}
// 	return ret, false
// }

// func (p *pull) Deactivate(gid game.Gid) []cgf.Event {
// 	ret := []cgf.Event{
// 		removePullEvent{
// 			PlayerGid: gid,
// 			Id:        p.id,
// 		},
// 	}
// 	return ret
// }

// type addPullEvent struct {
// 	PlayerGid game.Gid
// 	Id        int
// 	Angle     float64
// 	Force     float64
// 	Press     bool
// }

// func init() {
// 	gob.Register(addPullEvent{})
// }

// func (e addPullEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	player, ok := g.Ents[e.PlayerGid].(*game.Player)
// 	if !ok {
// 		return
// 	}
// 	if !e.Press {
// 		if proc := player.Processes[100+e.Id]; proc != nil {
// 			proc.Kill(g)
// 			delete(player.Processes, 100+e.Id)
// 		}
// 		return
// 	}
// 	player.Processes[100+e.Id] = &pullProcess{
// 		BasicPhases: BasicPhases{game.PhaseRunning},
// 		PlayerGid:   e.PlayerGid,
// 		Id:          e.Id,
// 		Angle:       e.Angle,
// 		Force:       e.Force,
// 	}
// }

// type removePullEvent struct {
// 	PlayerGid game.Gid
// 	Id        int
// }

// func init() {
// 	gob.Register(removePullEvent{})
// }

// func (e removePullEvent) Apply(_g interface{}) {
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
// 	gob.Register(&pullProcess{})
// }

// type pullProcess struct {
// 	BasicPhases
// 	NullCondition
// 	PlayerGid game.Gid
// 	Id        int
// 	Angle     float64
// 	Force     float64

// 	supplied float64
// }

// func (p *pullProcess) Supply(supply game.Mana) game.Mana {
// 	if supply[game.ColorBlue] > p.required()-p.supplied {
// 		supply[game.ColorBlue] -= p.required() - p.supplied
// 		p.supplied = p.required()
// 	} else {
// 		p.supplied += supply[game.ColorBlue]
// 		supply[game.ColorBlue] = 0
// 	}
// 	return supply
// }

// func (p *pullProcess) required() float64 {
// 	return p.Force / 10
// }

// func (p *pullProcess) reset() {
// 	p.supplied = 0
// }

// func (p *pullProcess) Think(g *game.Game) {
// 	defer p.reset()
// 	player, ok := g.Ents[p.PlayerGid].(*game.Player)
// 	if !ok {
// 		return
// 	}

// 	base_force := p.Force * p.supplied / p.required()
// 	g.DoForEnts(func(gid game.Gid, ent game.Ent) {
// 		if ent == game.Ent(player) {
// 			return
// 		}
// 		target_pos := ent.Pos()
// 		ray := target_pos.Sub(player.Pos())
// 		target_angle := ray.Angle() - player.Angle
// 		for target_angle < 0 {
// 			target_angle += math.Pi * 2
// 		}
// 		for target_angle > math.Pi*2 {
// 			target_angle -= math.Pi * 2
// 		}
// 		if target_angle > p.Angle/2 && target_angle < math.Pi*2-p.Angle/2 {
// 			return
// 		}
// 		ray = player.Pos().Sub(ent.Pos())
// 		// dist := ray.Mag()
// 		ray = ray.Norm()
// 		force := base_force // / math.Pow(dist, p.Angle/(2*math.Pi))
// 		ent.ApplyForce(ray.Scale(-force))
// 		player.ApplyForce(ray.Scale(force))
// 	})
// }

// func (p *pullProcess) Draw(gid game.Gid, g *game.Game, side int) {
// 	player, ok := g.Ents[p.PlayerGid].(*game.Player)
// 	if !ok {
// 		return
// 	}
// 	if side != player.Side() {
// 		return
// 	}
// 	gl.Color4d(1, 1, 1, 1)
// 	gl.Disable(gl.TEXTURE_2D)
// 	v1 := player.Pos()
// 	v2 := v1.Add(linear.Vec2{1000, 0})
// 	v3 := v2.RotateAround(v1, player.Angle-p.Angle/2)
// 	v4 := v2.RotateAround(v1, player.Angle+p.Angle/2)
// 	gl.Begin(gl.LINES)
// 	vs := []linear.Vec2{v3, v4, player.Pos()}
// 	for i := range vs {
// 		gl.Vertex2d(gl.Double(vs[i].X), gl.Double(vs[i].Y))
// 		gl.Vertex2d(gl.Double(vs[(i+1)%len(vs)].X), gl.Double(vs[(i+1)%len(vs)].Y))
// 	}
// 	gl.End()
// }
