package ability

// import (
// 	"encoding/gob"
// 	"github.com/runningwild/cgf"
// 	"github.com/runningwild/magnus/game"
// 	"math"
// )

// func makeVision(params map[string]int) game.Ability {
// 	var b vision
// 	b.id = NextAbilityId()
// 	b.distance = float64(params["range"])
// 	b.squeeze = float64(params["squeeze"]) / 1000
// 	return &b
// }

// func init() {
// 	game.RegisterAbility("vision", makeVision)
// }

// type vision struct {
// 	NonResponder
// 	NonThinker
// 	NonRendering

// 	id       int
// 	distance float64
// 	squeeze  float64
// }

// func (p *vision) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
// 	ret := []cgf.Event{
// 		addVisionEvent{
// 			PlayerGid: gid,
// 			Id:        p.id,
// 			Distance:  p.distance,
// 			Squeeze:   p.squeeze,
// 			Press:     keyPress,
// 		},
// 	}
// 	return ret, false
// }

// func (p *vision) Deactivate(gid game.Gid) []cgf.Event {
// 	return nil
// 	// ret := []cgf.Event{
// 	// 	removeVisionEvent{
// 	// 		PlayerGid: gid,
// 	// 		Id:        p.id,
// 	// 	},
// 	// }
// 	// return ret
// }

// type addVisionEvent struct {
// 	PlayerGid game.Gid
// 	Id        int
// 	Distance  float64
// 	Squeeze   float64
// 	Press     bool
// }

// func init() {
// 	gob.Register(addVisionEvent{})
// }

// func (e addVisionEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	player, ok := g.Ents[e.PlayerGid].(*game.Player)
//  if !ok {
//    return
//  }
// 	if !e.Press {
// 		if proc := player.Processes[100+e.Id]; proc != nil {
// 			proc.Kill(g)
// 			delete(player.Processes, 100+e.Id)
// 		}
// 		return
// 	}
// 	player.Processes[100+e.Id] = &visionProcess{
// 		BasicPhases: BasicPhases{game.PhaseRunning},
// 		Id:          e.Id,
// 		PlayerGid:   e.PlayerGid,
// 		Distance:    e.Distance,
// 		Squeeze:     e.Squeeze,
// 	}
// }

// type removeVisionEvent struct {
// 	PlayerGid game.Gid
// 	Id        int
// }

// func init() {
// 	gob.Register(removeVisionEvent{})
// }

// func (e removeVisionEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	player, ok := g.Ents[e.PlayerGid].(*game.Player)
//  if !ok {
//    return
//  }
// 	proc := player.Processes[100+e.Id]
// 	if proc != nil {
// 		proc.Kill(g)
// 		delete(player.Processes, 100+e.Id)
// 	}
// }

// func init() {
// 	gob.Register(&visionProcess{})
// }

// type visionProcess struct {
// 	BasicPhases
// 	NullCondition
// 	Id        int
// 	PlayerGid game.Gid
// 	Distance  float64
// 	Squeeze   float64

// 	supplied float64
// }

// const visionHorizon = 500

// func (p *visionProcess) Supply(supply game.Mana) game.Mana {
// 	if supply[game.ColorGreen] > p.required()-p.supplied {
// 		supply[game.ColorGreen] -= p.required() - p.supplied
// 		p.supplied = p.required()
// 	} else {
// 		p.supplied += supply[game.ColorGreen]
// 		supply[game.ColorGreen] = 0
// 	}
// 	return supply
// }

// const visionManaFactor = 0.001

// // Provides some estimate of how much mana should be required to cast vision
// // with parameters k, d, and maxDist
// func manaCostFromMaxDist(k, d, maxDist float64) float64 {
// 	return math.Pi * maxDist * maxDist * math.Pow(1+d, -k) * visionManaFactor
// }

// // C = pi * M^2 * (1+d) ^ (-k)
// // M = sqrt(C/(pi * (1+d) ^ (-k)))
// func maxDistFromManaCost(k, d, manaCost float64) float64 {
// 	return math.Sqrt(manaCost / (math.Pi * math.Pow(1+d, -k) * visionManaFactor))
// }

// // For parabola y=kx^2-d
// func dist(angle, k, d float64) float64 {
// 	sin := math.Sin(angle)
// 	cos := math.Cos(angle)
// 	cos2 := cos * cos
// 	inner := sin*sin - 4*(k*cos2)*(-d)
// 	if inner < 0 || cos == 0 {
// 		return math.Inf(1)
// 	}
// 	v := (sin + math.Sqrt(inner)) / (2 * k * cos2)
// 	return v
// }

// // Finds angle such that dist(angle, k, d) is minimized.
// func minDist(k, d float64) float64 {
// 	// Ternary search on dist, this way we don't have to take a horrifying
// 	// derivative just to do a binary search.
// 	max := math.Pi/2 - 0.1 // good enough for any sensible values of k and d.
// 	min := -math.Pi / 2
// 	high := max - (max-min)/3
// 	low := min + (max-min)/3
// 	for high-low > 1e-5 {
// 		rhigh := dist(high, k, d)
// 		rlow := dist(low, k, d)
// 		if rhigh < rlow {
// 			min = low
// 		} else {
// 			max = high
// 		}
// 		high = max - (max-min)/3
// 		low = min + (max-min)/3
// 	}
// 	return high
// }

// func (p *visionProcess) required() float64 {
// 	return manaCostFromMaxDist(p.Squeeze, p.Distance, visionHorizon)
// }

// func (p *visionProcess) reset() {
// 	p.supplied = 0
// }

// func (p *visionProcess) Think(g *game.Game) {
// 	defer p.reset()
// 	horizon := maxDistFromManaCost(p.Squeeze, p.Distance, p.supplied)
// 	player, ok := g.Ents[p.PlayerGid].(*game.Player)
//  if !ok {
//    return
//  }
// 	zbuffer := player.Los.RawAccess()
// 	for i := range zbuffer {
// 		bufferAngle := float64(i) / float64(len(zbuffer)) * 2 * math.Pi
// 		angle := player.Angle - bufferAngle - math.Pi/2
// 		d := dist(angle, p.Squeeze, p.Distance)
// 		if d > horizon {
// 			d = horizon
// 		}
// 		d = d * d
// 		if float64(zbuffer[i]) < d {
// 			zbuffer[i] = float32(d)
// 		}
// 	}
// }

// func (p *visionProcess) Draw(gid game.Gid, g *game.Game) {
// }
