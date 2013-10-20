package ability

// import (
// 	"encoding/gob"
// 	gl "github.com/chsc/gogl/gl21"
// 	"github.com/runningwild/cgf"
// 	"github.com/runningwild/glop/gin"
// 	"github.com/runningwild/jota/base"
// 	"github.com/runningwild/jota/game"
// 	"github.com/runningwild/jota/texture"
// 	"github.com/runningwild/linear"
// 	"math"
// 	"math/rand"
// )

// func makePlaceMine(params map[string]int) game.Ability {
// 	var pm placeMine
// 	pm.id = NextAbilityId()
// 	pm.health = float64(params["health"])
// 	pm.damage = float64(params["damage"])
// 	pm.trigger = float64(params["trigger"])
// 	pm.mass = float64(params["mass"])
// 	pm.cost = float64(params["cost"])
// 	return &pm
// }

// func init() {
// 	game.RegisterAbility("mine", makePlaceMine)
// }

// type placeMine struct {
// 	NonRendering

// 	id      int
// 	health  float64
// 	damage  float64
// 	trigger float64
// 	mass    float64
// 	cost    float64
// 	fire    int
// }

// func (pm *placeMine) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
// 	var ret []cgf.Event
// 	if keyPress {
// 		ret = append(ret, addPlaceMineCastProcessEvent{
// 			PlayerGid: gid,
// 			ProcessId: pm.id,
// 			Cost:      pm.cost,
// 		})
// 	} else {
// 		ret = append(ret, removePlaceMineCastProcessEvent{
// 			PlayerGid: gid,
// 			ProcessId: pm.id,
// 		})
// 	}
// 	return ret, keyPress
// }

// func (pm *placeMine) Deactivate(gid game.Gid) []cgf.Event {
// 	pm.fire = 0
// 	return nil
// }

// func (pm *placeMine) Respond(gid game.Gid, group gin.EventGroup) bool {
// 	if found, event := group.FindEvent(gin.AnySpace); found && event.Type == gin.Press {
// 		pm.fire++
// 		return true
// 	}
// 	return false
// }

// func (pm *placeMine) Think(gid game.Gid, g *game.Game) ([]cgf.Event, bool) {
// 	var ret []cgf.Event
// 	for ; pm.fire > 0; pm.fire-- {
// 		ret = append(ret, addPlaceMineFireEvent{
// 			PlayerGid: gid,
// 			ProcessId: pm.id,
// 			Health:    pm.health,
// 			Mass:      pm.mass,
// 			Damage:    pm.damage,
// 			Trigger:   pm.trigger,
// 		})
// 	}
// 	return ret, false
// }

// type placeMineCastProcess struct {
// 	BasicPhases
// 	NullCondition
// 	PlayerGid game.Gid
// 	Stored    game.Mana
// 	Cost      float64
// }

// func (p *placeMineCastProcess) Supply(supply game.Mana) game.Mana {
// 	p.Stored[game.ColorBlue] += supply[game.ColorBlue]
// 	supply[game.ColorBlue] = 0
// 	return supply
// }

// func (p *placeMineCastProcess) Think(g *game.Game) {
// 	p.Stored[game.ColorBlue] *= 0.98
// }

// // TODO: This function really needs to take not just the side, but the player
// // that this is being drawn for.
// func (p *placeMineCastProcess) Draw(gid game.Gid, g *game.Game, side int) {
// 	player, _ := g.Ents[p.PlayerGid].(*game.PlayerEnt)
// 	if player == nil {
// 		return
// 	}
// 	if side != player.Side() {
// 		return
// 	}
// 	ready := int(p.Stored[game.ColorBlue] / p.Cost)
// 	base.EnableShader("status_bar")
// 	if ready == 0 {
// 		gl.Color4ub(255, 0, 0, 255)
// 	} else {
// 		gl.Color4ub(0, 255, 0, 255)
// 	}
// 	var outer float32 = 0.2
// 	var increase float32 = 0.01
// 	frac := p.Stored[game.ColorBlue] / p.Cost
// 	base.SetUniformF("status_bar", "frac", float32(frac-float64(ready)))
// 	base.SetUniformF("status_bar", "inner", outer-increase*float32(ready+1))
// 	base.SetUniformF("status_bar", "outer", outer)
// 	base.SetUniformF("status_bar", "buffer", 0.01)
// 	texture.Render(player.Pos().X-100, player.Pos().Y-100, 200, 200)
// 	if ready > 0 {
// 		base.SetUniformF("status_bar", "frac", 1.0)
// 		base.SetUniformF("status_bar", "inner", outer-float32(ready)*increase)
// 		base.SetUniformF("status_bar", "outer", outer)
// 		texture.Render(player.Pos().X-100, player.Pos().Y-100, 200, 200)
// 	}
// 	base.EnableShader("")
// }

// type addPlaceMineCastProcessEvent struct {
// 	PlayerGid game.Gid
// 	ProcessId int
// 	Cost      float64
// }

// func init() {
// 	gob.Register(addPlaceMineCastProcessEvent{})
// }

// func (e addPlaceMineCastProcessEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	player, ok := g.Ents[e.PlayerGid].(*game.PlayerEnt)
// 	if !ok {
// 		return
// 	}
// 	player.Processes[100+e.ProcessId] = &placeMineCastProcess{
// 		PlayerGid: e.PlayerGid,
// 		Cost:      e.Cost,
// 	}
// }

// type removePlaceMineCastProcessEvent struct {
// 	PlayerGid game.Gid
// 	ProcessId int
// 	Cost      float64
// }

// func init() {
// 	gob.Register(removePlaceMineCastProcessEvent{})
// }

// func (e removePlaceMineCastProcessEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	player, ok := g.Ents[e.PlayerGid].(*game.PlayerEnt)
// 	if !ok {
// 		return
// 	}
// 	proc := player.Processes[100+e.ProcessId]
// 	if proc != nil {
// 		proc.Kill(g)
// 	}
// }

// type addPlaceMineFireEvent struct {
// 	PlayerGid game.Gid
// 	ProcessId int
// 	Health    float64
// 	Mass      float64
// 	Damage    float64
// 	Trigger   float64
// }

// func init() {
// 	gob.Register(addPlaceMineFireEvent{})
// }

// func (e addPlaceMineFireEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	player, ok := g.Ents[e.PlayerGid].(*game.PlayerEnt)
// 	if !ok {
// 		return
// 	}
// 	proc := player.Processes[100+e.ProcessId]
// 	if proc == nil {
// 		return
// 	}
// 	pmProc, ok := proc.(*placeMineCastProcess)
// 	if !ok {
// 		return
// 	}
// 	if pmProc.Stored[game.ColorBlue] < pmProc.Cost {
// 		// Can't cast until you've stored up the minimum amount
// 		return
// 	}
// 	pmProc.Stored[game.ColorBlue] -= pmProc.Cost

// 	var angle float64
// 	if player.Velocity.Mag() < 10 {
// 		angle = player.Velocity.Angle()
// 	} else {
// 		angle = player.Angle()
// 	}
// 	pos := player.Position.Add((linear.Vec2{50, 0}).Rotate(angle + math.Pi))
// 	rng := rand.New(g.Rng)
// 	pos = pos.Add((linear.Vec2{rng.NormFloat64() * 15, 0}).Rotate(rng.Float64() * math.Pi * 2))
// 	g.MakeMine(pos, player.Velocity.Scale(0.5), e.Health, e.Mass, e.Damage, e.Trigger)
// }
