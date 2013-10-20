package game

import (
	"github.com/runningwild/cgf"
)

// An Ability represents something a player can do that does not directly affect
// the game state.
type Ability interface {
	// Called any time the pressedness of either of the two keys bound to this
	// ability change.  Returns a list of events to apply.
	Input(gid Gid, pressAmt0, pressAmt1 float64) []cgf.Event

	// Returns any number of events to apply.  Typically this will include an
	// event that will add a Process to this player.
	Think(gid Gid, game *Game) []cgf.Event

	// If it is the active Ability it might want to draw some Ui stuff.
	Draw(gid Gid, game *Game)
}

type AbilityMaker func(params map[string]int) Ability

var ability_makers map[string]AbilityMaker

func RegisterAbility(name string, maker AbilityMaker) {
	if ability_makers == nil {
		ability_makers = make(map[string]AbilityMaker)
	}
	ability_makers[name] = maker
}
