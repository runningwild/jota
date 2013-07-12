package ability

import (
	"encoding/gob"
	"github.com/runningwild/cgf"
	"github.com/runningwild/magnus/game"
	"math"
)

func makeVision(params map[string]int) game.Ability {
	var b vision
	b.id = nextAbilityId()
	b.distance = float64(params["range"])
	b.squeeze = float64(params["squeeze"]) / 100
	return &b
}

func init() {
	game.RegisterAbility("vision", makeVision)
}

type vision struct {
	nonResponder
	nonThinker
	nonRendering

	id       int
	distance float64
	squeeze  float64
}

func (p *vision) Activate(player_id int, keyPress bool) ([]cgf.Event, bool) {
	ret := []cgf.Event{
		addVisionEvent{
			Player_id: player_id,
			Id:        p.id,
			Distance:  p.distance,
			Squeeze:   p.squeeze,
			Press:     keyPress,
		},
	}
	return ret, false
}

func (p *vision) Deactivate(player_id int) []cgf.Event {
	return nil
	// ret := []cgf.Event{
	// 	removeVisionEvent{
	// 		Player_id: player_id,
	// 		Id:        p.id,
	// 	},
	// }
	// return ret
}

type addVisionEvent struct {
	Player_id int
	Id        int
	Distance  float64
	Squeeze   float64
	Press     bool
}

func init() {
	gob.Register(addVisionEvent{})
}

func (e addVisionEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player := g.GetEnt(e.Player_id).(*game.Player)
	if !e.Press {
		if proc := player.Processes[100+e.Id]; proc != nil {
			proc.Kill(g)
			delete(player.Processes, 100+e.Id)
		}
		return
	}
	player.Processes[100+e.Id] = &visionProcess{
		BasicPhases: BasicPhases{game.PhaseRunning},
		Id:          e.Id,
		Player_id:   e.Player_id,
		Distance:    e.Distance,
		Squeeze:     e.Squeeze,
	}
}

type removeVisionEvent struct {
	Player_id int
	Id        int
}

func init() {
	gob.Register(removeVisionEvent{})
}

func (e removeVisionEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player := g.GetEnt(e.Player_id).(*game.Player)
	proc := player.Processes[100+e.Id]
	if proc != nil {
		proc.Kill(g)
		delete(player.Processes, 100+e.Id)
	}
}

func init() {
	gob.Register(&visionProcess{})
}

type visionProcess struct {
	BasicPhases
	NullCondition
	Id        int
	Player_id int
	Distance  float64
	Squeeze   float64

	required float64
	supplied float64
}

func (p *visionProcess) Copy() game.Process {
	p2 := *p
	return &p2
}

func (p *visionProcess) PreThink(g *game.Game) {
	p.required = p.Distance
	p.supplied = 0
}
func (p *visionProcess) Supply(supply game.Mana) game.Mana {
	return supply
	// if supply[game.ColorBlue] > p.required-p.supplied {
	// 	supply[game.ColorBlue] -= p.required - p.supplied
	// 	p.supplied = p.required
	// } else {
	// 	p.supplied += supply[game.ColorBlue]
	// 	supply[game.ColorBlue] = 0
	// }
	// return supply
}

// For parabola y=kx^2-d
func dist(angle, k, d float64) float64 {
	sin := math.Sin(angle)
	cos := math.Cos(angle)
	inner := sin*sin - 4*(k*cos*cos)*(-d)
	if inner < 0 || cos == 0 {
		return math.Inf(1)
	}
	v := (sin + math.Sqrt(inner)) / (2 * k * cos * cos)
	return v
}

func (p *visionProcess) Think(g *game.Game) {
	_player := g.GetEnt(p.Player_id)
	player := _player.(*game.Player)
	zbuffer := player.Los.RawAccess()
	for i := range zbuffer {
		bufferAngle := float64(i) / float64(len(zbuffer)) * 2 * math.Pi
		angle := player.Angle - bufferAngle - math.Pi/2
		d := dist(angle, 0.01, 50)
		if d > 500 {
			d = 500
		}
		d = d * d
		if float64(zbuffer[i]) < d {
			zbuffer[i] = float32(d)
		}
	}
}

func (p *visionProcess) Draw(player_id int, g *game.Game) {
}
