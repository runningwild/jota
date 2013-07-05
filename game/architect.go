package game

type architectData struct {
	// Resources available to spend
	Value int

	// Resources replenished each frame
	Restore int
}

// Called as part of Game.Think()
func (arch *architectData) Think(g *Game) {

}
