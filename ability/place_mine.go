package ability

import (
	"encoding/gob"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/linear"
)

func makePlaceMine(params map[string]int) game.Ability {
	var pm placeMine
	pm.id = NextAbilityId()
	pm.health = float64(params["health"])
	pm.damage = float64(params["damage"])
	pm.trigger = float64(params["trigger"])
	pm.mass = float64(params["mass"])
	pm.cost = float64(params["cost"])
	return &pm
}

func init() {
	game.RegisterAbility("mine", makePlaceMine)
	gob.Register(&placeMine{})
}

type placeMine struct {
	id      int
	health  float64
	damage  float64
	trigger float64
	mass    float64
	cost    float64
	fire    int
}

func (pm *placeMine) Input(ent game.Ent, g *game.Game, pressAmt float64, trigger bool) {
	player := ent.(*game.PlayerEnt)
	if pressAmt == 0 {
		delete(player.Processes, pm.id)
		return
	}
	proc, ok := player.Processes[pm.id].(*multiDrain)
	if !ok {
		player.Processes[pm.id] = &multiDrain{Gid: player.Gid, Unit: game.Mana{300, 0, 0}}
		return
	}
	if trigger && proc.Stored > 1 {
		proc.Stored--
		heading := (linear.Vec2{1, 0}).Rotate(ent.Angle())
		pos := ent.Pos().Add(heading.Scale(100))
		g.MakeMine(pos, linear.Vec2{}, 100, 100, 100, 100)
	}
}
func (pm *placeMine) Think(ent game.Ent, game *game.Game) {

}
func (pm *placeMine) Draw(ent game.Ent, game *game.Game) {

}
func (pm *placeMine) IsActive() bool {
	return false
}
