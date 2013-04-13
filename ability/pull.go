package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	// "github.com/runningwild/glop/gin"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/game"
	"math"
)

var pull_id int

func makePull(params map[string]int) game.Ability {
	var b pull
	b.id = pull_id
	b.force = float64(params["force"])
	b.angle = float64(params["angle"]) / 180 * 3.14159
	pull_id++
	return &b
}

func init() {
	game.RegisterAbility("pull", makePull)
}

type pull struct {
	nonResponder
	nonThinker
	nonRendering

	id    int
	force float64
	angle float64
}

func (p *pull) Activate(player_id int) ([]cgf.Event, bool) {
	ret := []cgf.Event{
		addPullEvent{
			Player_id: player_id,
			Id:        p.id,
			Force:     p.force,
			Angle:     p.angle,
		},
	}
	return ret, true
}

func (p *pull) Deactivate(player_id int) []cgf.Event {
	ret := []cgf.Event{
		removePullEvent{
			Player_id: player_id,
			Id:        p.id,
		},
	}
	return ret
}

func (p *pull) Think(player_id int, g *game.Game) ([]cgf.Event, bool) {
	return nil, true
	// if gin.In().GetKey(gin.AnyEscape).FramePressCount() > 0 {
	// 	return nil, true
	// }

	// player := g.GetEnt(player_id).(*game.Player)
	// // mx, my := gin.In().GetCursor("Mouse").Point()
	// mx, my := 1, 2
	// rx := g.Region().Point.X
	// ry := g.Region().Point.Y
	// var v1, v2 linear.Vec2
	// v1.X = player.X
	// v1.Y = player.Y
	// v2.X = float64(mx - rx)
	// v2.Y = float64(my - ry)
	// p.x = v2.X
	// p.y = v2.Y

	// ret := []cgf.Event{
	// 	removePullEvent{
	// 		Player_id: player_id,
	// 		Id:        p.id,
	// 	},
	// }
	// if gin.In().GetKey(gin.AnyMouseLButton).FramePressAmt() > 0 {
	// 	ret = append(ret, addPullEvent{
	// 		Player_id: player_id,
	// 		Id:        p.id,
	// 		X:         p.x,
	// 		Y:         p.y,
	// 		Angle:     p.angle,
	// 		Force:     p.force,
	// 	})
	// }
	// return ret, false
}

func (p *pull) Draw(player_id int, g *game.Game) {
	player := g.GetEnt(player_id).(*game.Player)
	v1 := player.Pos()
	var v2 linear.Vec2
	// v2.X = p.x
	// v2.Y = p.y
	v2 = v2.Sub(v1).Norm().Scale(1000).Add(v1)
	v3 := v2.RotateAround(v1, p.angle)
	v4 := v2.RotateAround(v1, -p.angle)
	gl.Begin(gl.LINES)
	vs := []linear.Vec2{v3, v4, linear.Vec2{player.X, player.Y}}
	for i := range vs {
		gl.Vertex2d(gl.Double(vs[i].X), gl.Double(vs[i].Y))
		gl.Vertex2d(gl.Double(vs[(i+1)%len(vs)].X), gl.Double(vs[(i+1)%len(vs)].Y))
	}
	gl.End()
}

type addPullEvent struct {
	Player_id int
	Id        int
	X, Y      float64
	Angle     float64
	Force     float64
}

func init() {
	gob.Register(addPullEvent{})
}

func (e addPullEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player := g.GetEnt(e.Player_id).(*game.Player)
	player.Processes[100+e.Id] = &pullProcess{
		BasicPhases: BasicPhases{game.PhaseRunning},
		Id:          e.Id,
		Player_id:   e.Player_id,
		X:           e.X,
		Y:           e.Y,
		Angle:       e.Angle,
		Force:       e.Force,
	}
}

type removePullEvent struct {
	Player_id int
	Id        int
}

func init() {
	gob.Register(removePullEvent{})
}

func (e removePullEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player := g.GetEnt(e.Player_id).(*game.Player)
	proc := player.Processes[100+e.Id]
	if proc != nil {
		proc.Kill(g)
		delete(player.Processes, 100+e.Id)
	}
}

func init() {
	gob.Register(&pullProcess{})
}

type pullProcess struct {
	BasicPhases
	NullCondition
	Id        int
	Player_id int
	X, Y      float64
	Angle     float64
	Force     float64

	required float64
	supplied float64
}

func (p *pullProcess) PreThink(g *game.Game) {
	p.required = p.Force
	p.supplied = 0
}
func (p *pullProcess) Supply(supply game.Mana) game.Mana {
	if supply[game.ColorBlue] > p.required-p.supplied {
		supply[game.ColorBlue] -= p.required - p.supplied
		p.supplied = p.required
	} else {
		p.supplied += supply[game.ColorBlue]
		supply[game.ColorBlue] = 0
	}
	return supply
}
func (p *pullProcess) Think(g *game.Game) {
	_player := g.GetEnt(p.Player_id)
	player := _player.(*game.Player)
	source_pos := linear.Vec2{p.X, p.Y}

	base_force := p.Force * p.supplied / p.required
	for _, _target := range g.Ents {
		target, ok := _target.(*game.Player)
		if !ok || target == player {
			continue
		}
		target_pos := linear.Vec2{target.X, target.Y}
		ray := target_pos.Sub(player.Pos())
		target_angle := ray.Angle()
		process_angle := source_pos.Sub(player.Pos()).Angle()
		angle := target_angle - process_angle
		if angle < 0 {
			angle = -angle
		}
		if angle > p.Angle {
			continue
		}
		ray = player.Pos().Sub(target.Pos())
		dist := ray.Mag()
		ray = ray.Norm()
		force := base_force / math.Pow(dist, p.Angle/(2*math.Pi))
		target.ApplyForce(ray.Scale(-force))
		player.ApplyForce(ray.Scale(force))
	}
}

func (p *pullProcess) Draw(player_id int, g *game.Game) {
}
