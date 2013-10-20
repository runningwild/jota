package ability

// import (
// 	"encoding/gob"
// 	"github.com/runningwild/cgf"
// 	"github.com/runningwild/jota/game"
// 	"github.com/runningwild/jota/stats"
// )

// func makeCloak(params map[string]int) game.Ability {
// 	var c cloak
// 	c.id = NextAbilityId()
// 	return &c
// }

// func init() {
// 	game.RegisterAbility("cloak", makeCloak)
// }

// type cloak struct {
// 	NonResponder
// 	NonThinker
// 	NonRendering

// 	id int
// }

// func (p *cloak) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
// 	ret := []cgf.Event{
// 		addCloakEvent{
// 			PlayerGid: gid,
// 			Id:        p.id,
// 			Press:     keyPress,
// 		},
// 	}
// 	return ret, false
// }

// func (p *cloak) Deactivate(gid game.Gid) []cgf.Event {
// 	return nil
// 	ret := []cgf.Event{
// 		removeCloakEvent{
// 			PlayerGid: gid,
// 			Id:        p.id,
// 		},
// 	}
// 	return ret
// }

// type addCloakEvent struct {
// 	PlayerGid game.Gid
// 	Id        int
// 	Press     bool
// }

// func init() {
// 	gob.Register(addCloakEvent{})
// }

// func (e addCloakEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	player, ok := g.Ents[e.PlayerGid].(*game.PlayerEnt)
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
// 	player.Processes[100+e.Id] = &cloakProcess{
// 		BasicPhases: BasicPhases{game.PhaseRunning},
// 		Id:          e.Id,
// 		PlayerGid:   e.PlayerGid,
// 	}
// }

// type removeCloakEvent struct {
// 	PlayerGid game.Gid
// 	Id        int
// }

// func init() {
// 	gob.Register(removeCloakEvent{})
// }

// func (e removeCloakEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	player, ok := g.Ents[e.PlayerGid].(*game.PlayerEnt)
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
// 	gob.Register(&cloakProcess{})
// }

// type cloakProcess struct {
// 	BasicPhases
// 	NullCondition
// 	NonRendering
// 	Id        int
// 	PlayerGid game.Gid

// 	supplied float64
// 	alpha    float64
// }

// func (p *cloakProcess) ModifyBase(base stats.Base) stats.Base {
// 	base.Cloaking = 1.0 - ((1.0 - base.Cloaking) * p.alpha)
// 	return base
// }

// const cloakRate = 15

// func (p *cloakProcess) Supply(supply game.Mana) game.Mana {
// 	if supply[game.ColorGreen] > cloakRate-p.supplied {
// 		supply[game.ColorGreen] -= cloakRate - p.supplied
// 		p.supplied = cloakRate
// 	} else {
// 		p.supplied += supply[game.ColorGreen]
// 		supply[game.ColorGreen] = 0
// 	}
// 	return supply
// }

// func (p *cloakProcess) Think(g *game.Game) {
// 	p.alpha = 1.0 - float64(p.supplied)/float64(cloakRate)
// 	p.supplied = 0
// }
