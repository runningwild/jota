package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
)

var pull_id int

func makePull(params map[string]int) game.Ability {
	var b pull
	b.id = pull_id
	b.force = 250
	b.angle = 0.15
	pull_id++
	return &b
}

func init() {
	game.RegisterAbility("pull", makePull)
}

type pull struct {
	nonResponder
	nonThinker

	id    int
	force float64
	x, y  float64
	angle float64
}

func (p *pull) Activate(player_id int) ([]cgf.Event, bool) {
	return nil, true
}

func (p *pull) Deactivate(player_id int) []cgf.Event {
	return nil
}

func (p *pull) Think(player_id int, g *game.Game) ([]cgf.Event, bool) {
	if gin.In().GetKey(gin.Escape).FramePressCount() > 0 {
		return nil, true
	}

	player := g.GetEnt(player_id).(*game.Player)
	mx, my := gin.In().GetCursor("Mouse").Point()
	rx := g.Region().Point.X
	ry := g.Region().Point.Y
	var v1, v2 linear.Vec2
	v1.X = player.X
	v1.Y = player.Y
	v2.X = float64(mx - rx)
	v2.Y = float64(my - ry)
	p.x = v2.X
	p.y = v2.Y

	ret := []cgf.Event{
		removePullEvent{
			Player_id: player_id,
			Id:        p.id,
		},
	}
	if gin.In().GetKey(gin.MouseLButton).FramePressAmt() > 0 {
		base.Log().Printf("Add event")
		ret = append(ret, addPullEvent{
			Player_id: player_id,
			Id:        p.id,
			X:         p.x,
			Y:         p.y,
			Angle:     p.angle,
			Force:     p.force,
		})
	}
	return ret, false
}

func (p *pull) Draw(player_id int, g *game.Game) {
	player := g.GetEnt(player_id).(*game.Player)
	var v1, v2 linear.Vec2
	v1.X = player.X
	v1.Y = player.Y
	v2.X = p.x
	v2.Y = p.y
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

func (e addPullEvent) ApplyFirst(g interface{}) {}
func (e addPullEvent) ApplyFinal(g interface{}) {}
func (e addPullEvent) Apply(_g interface{}) {
	base.Log().Printf("Add pull process")
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

func (e removePullEvent) ApplyFirst(g interface{}) {}
func (e removePullEvent) ApplyFinal(g interface{}) {}
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

	Gathered game.Mana
	required float64
	supplied float64
	targets  []*game.Player
	com_mass float64
	com_pos  linear.Vec2
}

func (p *pullProcess) PreThink(g *game.Game) {
	p.required = 0
	p.supplied = 0
	p.com_mass = 0
	p.com_pos = linear.Vec2{}
	_player := g.GetEnt(p.Player_id)
	player := _player.(*game.Player)
	player_pos := linear.Vec2{player.X, player.Y}
	max_dist_sq := player_pos.Sub(linear.Vec2{p.X, p.Y}).Mag2()
	base.Log().Printf("p: %v", p)
	base.Log().Printf("pos: %v", player_pos)
	for _, _target := range g.Ents {
		target, ok := _target.(*game.Player)
		if !ok || target == player {
			continue
		}
		target_pos := linear.Vec2{target.X, target.Y}
		ray := target_pos.Sub(player_pos)
		dist_sq := ray.Mag2()
		if dist_sq > max_dist_sq {
			continue
		}
		target_angle := ray.Angle()
		process_angle := linear.Vec2{p.X, p.Y}.Sub(player_pos).Angle()
		angle := target_angle - process_angle
		if angle < 0 {
			angle = -angle
		}
		if angle > p.Angle {
			continue
		}
		p.targets = append(p.targets, target)
	}

	if len(p.targets) == 0 {
		return
	}

	for _, target := range p.targets {
		target_pos := linear.Vec2{target.X, target.Y}
		dist_sq := player_pos.Sub(target_pos).Mag2()
		base.Log().Printf("Dist: %v\n", dist_sq)
		p.required += p.Force * dist_sq / 100000
	}

	for _, target := range p.targets {
		p.com_pos.X += target.X * target.Mass()
		p.com_pos.Y += target.Y * target.Mass()
		p.com_mass += target.Mass()
	}
	base.Log().Printf("Required: %v\n", p.required)
	base.Log().Printf("Pcommass %v", p.com_mass)
	p.com_pos = p.com_pos.Scale(1 / p.com_mass)
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
	if len(p.targets) == 0 {
		return
	}
	base.Log().Printf("%d %v\n", len(p.targets), p.required)
	_player := g.GetEnt(p.Player_id)
	player := _player.(*game.Player)
	player_pos := linear.Vec2{player.X, player.Y}
	force := p.Force * p.supplied / p.required
	ray := player_pos.Sub(p.com_pos)
	player.ApplyForce(ray.Norm().Scale(-force * float64(len(p.targets))))

	for _, target := range p.targets {
		ray := player_pos.Sub(linear.Vec2{target.X, target.Y})
		ray = ray.Norm()
		target.ApplyForce(ray.Scale(force))
	}
	base.Log().Printf("Required: %v\n", p.required)
}

func (p *pullProcess) Draw(player_id int, g *game.Game) {
}
