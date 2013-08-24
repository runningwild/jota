package ability

import (
	"encoding/gob"
	"fmt"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/magnus/stats"
)

func makeCloak(params map[string]int) game.Ability {
	var c cloak
	c.id = nextAbilityId()
	return &c
}

func init() {
	game.RegisterAbility("cloak", makeCloak)
}

type cloak struct {
	nonResponder
	nonThinker
	nonRendering

	id int
}

func (p *cloak) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	ret := []cgf.Event{
		addCloakEvent{
			PlayerGid: gid,
			Id:        p.id,
			Press:     keyPress,
		},
	}
	return ret, false
}

func (p *cloak) Deactivate(gid game.Gid) []cgf.Event {
	return nil
	ret := []cgf.Event{
		removeCloakEvent{
			PlayerGid: gid,
			Id:        p.id,
		},
	}
	return ret
}

type addCloakEvent struct {
	PlayerGid game.Gid
	Id        int
	Press     bool
}

func init() {
	gob.Register(addCloakEvent{})
}

func (e addCloakEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player := g.Ents[e.PlayerGid].(*game.Player)
	if !e.Press {
		if proc := player.Processes[100+e.Id]; proc != nil {
			proc.Kill(g)
			delete(player.Processes, 100+e.Id)
		}
		return
	}
	player.Processes[100+e.Id] = &cloakProcess{
		BasicPhases: BasicPhases{game.PhaseRunning},
		Id:          e.Id,
		PlayerGid:   e.PlayerGid,
	}
}

type removeCloakEvent struct {
	PlayerGid game.Gid
	Id        int
}

func init() {
	gob.Register(removeCloakEvent{})
}

func (e removeCloakEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player := g.Ents[e.PlayerGid].(*game.Player)
	proc := player.Processes[100+e.Id]
	if proc != nil {
		proc.Kill(g)
		delete(player.Processes, 100+e.Id)
	}
}

func init() {
	gob.Register(&cloakProcess{})
}

type cloakProcess struct {
	BasicPhases
	NullCondition
	Id        int
	PlayerGid game.Gid

	supplied float64
	alpha    float64
}

func (p *cloakProcess) ModifyBase(base stats.Base) stats.Base {
	base.Cloaking = 1.0 - ((1.0 - base.Cloaking) * p.alpha)
	return base
}

const cloakRate = 15

func (p *cloakProcess) Supply(supply game.Mana) game.Mana {
	if supply[game.ColorBlue] > cloakRate-p.supplied {
		supply[game.ColorBlue] -= cloakRate - p.supplied
		p.supplied = cloakRate
	} else {
		p.supplied += supply[game.ColorBlue]
		supply[game.ColorBlue] = 0
	}
	return supply
}

func (p *cloakProcess) Think(g *game.Game) {
	p.alpha = 1.0 - float64(p.supplied)/float64(cloakRate)
	p.supplied = 0
}

func (p *cloakProcess) Draw(gid game.Gid, g *game.Game) {
	player := g.Ents[p.PlayerGid].(*game.Player)
	base.GetDictionary("luxisr").RenderString(fmt.Sprintf("%v", player.Stats().Cloaking()), 100, 100, 0, 100, gui.Left)
}
