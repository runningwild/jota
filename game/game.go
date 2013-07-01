package game

import (
	"encoding/gob"
	//"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/util/algorithm"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/los"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
	"math"
	"path/filepath"
)

// type Ability func(game *Game, player *Player, params map[string]int) Process

// An Ability represents something a player can do that does not directly affect
// the game state.
type Ability interface {
	// Called when a player selects this Ability.  Returns any number of events to
	// apply, as well as a bool that is true iff this Ability should become the
	// active Ability.
	Activate(player_id int) ([]cgf.Event, bool)

	// Called when this Ability is deselected.
	Deactivate(player_id int) []cgf.Event

	// The active Ability will receive all of the events from the player.  It
	// should return true iff it consumes the event.
	Respond(player_id int, group gin.EventGroup) bool

	// Returns any number of events to apply, as well as a bool that is true iff
	// this Ability should be deactivated.  Typically this will include an event
	// that will add a Process to this player.
	Think(player_id int, game *Game) ([]cgf.Event, bool)

	// If it is the active Ability it might want to draw some Ui stuff.
	Draw(player_id int, game *Game)
}

type AbilityMaker func(params map[string]int) Ability

var ability_makers map[string]AbilityMaker

func RegisterAbility(name string, maker AbilityMaker) {
	if ability_makers == nil {
		ability_makers = make(map[string]AbilityMaker)
	}
	ability_makers[name] = maker
}

type Drain interface {
	// Supplies mana to the Process and returns the unused portion.
	Supply(Mana) Mana
}

type Phase int

const (
	// This phase is for any process that needs a ui before.  A player can only
	// have one Process in PhaseUi at a time.  If a player tries to use an ability
	// while a Process is in PhaseUi the process in PhaseUi will be killed.
	PhaseUi Phase = iota

	// Once a Process hits PhaseRunning it will remain here until it is complete.
	// A process should not reach this phase until it is done with player
	// interaction.
	PhaseRunning

	// Once a Process returns PhaseComplete it will always return PhaseComplete.
	PhaseComplete
)

type Thinker interface {
	PreThink(game *Game)
	Think(game *Game)

	// Kills a process.  Any Killed process will return true on any future
	// calls to Complete().
	Kill(game *Game)

	Phase() Phase
}

// TODO: Might want to be able to respond to events directly for Ui stuff
type Responder interface {
}

type Process interface {
	Drain
	Thinker
	Responder
	stats.Condition
	Draw(player_id int, game *Game)
	Copy() Process
}

type Color int

const (
	ColorRed Color = iota
	ColorGreen
	ColorBlue
)

func init() {
}

type Player struct {
	Stats  stats.Inst
	X, Y   float64
	Vx, Vy float64
	Angle  float64
	Delta  struct {
		Speed float64
		Angle float64
	}
	Color struct {
		R, G, B byte
	}

	// Unique Id over all entities ever
	Gid int

	// If Exile_frames > 0 then the Player is not present in the game right now
	// and is excluded from all combat/mana/rendering/processing/etc...
	// Exile_frames is the number of frames remaining that the player is in
	// exile.
	Exile_frames int32

	// Processes contains all of the processes that this player is casting
	// right now.
	Processes map[int]Process

	Los *los.Los
}

func (p *Player) Copy() Ent {
	p2 := *p
	p2.Processes = make(map[int]Process)
	for k, v := range p.Processes {
		p2.Processes[k] = v.Copy()
		if v == nil {
			panic("ASDF")
		}
	}
	p2.Los = p.Los.Copy()
	return &p2
}

func init() {
	gob.Register(&Player{})
}

func (p *Player) Alive() bool {
	return p.Stats.HealthCur() > 0
}

func (p *Player) Exiled() bool {
	return p.Exile_frames > 0
}

func (p *Player) ApplyForce(f linear.Vec2) {
	dv := f.Scale(1 / p.Mass())
	p.Vx += dv.X
	p.Vy += dv.Y
}

func (p *Player) ApplyDamage(d stats.Damage) {
	p.Stats.ApplyDamage(d)
}

func (p *Player) Mass() float64 {
	return p.Stats.Mass()
}

func (p *Player) Id() int {
	return p.Gid
}

func (p *Player) SetId(id int) {
	p.Gid = id
}

func (p *Player) Pos() linear.Vec2 {
	return linear.Vec2{p.X, p.Y}
}

func (p *Player) Vel() linear.Vec2 {
	return linear.Vec2{p.Vx, p.Vy}
}

func (p *Player) SetPos(pos linear.Vec2) {
	p.X = pos.X
	p.Y = pos.Y
}

func (p *Player) SetVel(vel linear.Vec2) {
	p.X = vel.X
	p.Y = vel.Y
}

func (p *Player) Draw(game *Game) {
	if p.Exiled() {
		return
	}
	var t *texture.Data
	gl.Color4ub(255, 255, 255, 255)
	if p.Id() == 1 {
		t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship.png"))
	} else if p.Id() == 2 {
		t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship3.png"))
	} else {
		t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship2.png"))
	}
	t.RenderAdvanced(p.X-float64(t.Dx())/2, p.Y-float64(t.Dy())/2, float64(t.Dx()), float64(t.Dy()), p.Angle, false)

	for _, proc := range p.Processes {
		proc.Draw(p.Id(), game)
	}
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.08)
	base.SetUniformF("status_bar", "outer", 0.09)
	base.SetUniformF("status_bar", "buffer", 0.01)

	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(125, 125, 125, 100)
	texture.Render(p.X-100, p.Y-100, 200, 200)

	health_frac := float32(p.Stats.HealthCur() / p.Stats.HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, 255)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, 255)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(p.X-100, p.Y-100, 200, 200)
	base.EnableShader("")
}

func (p *Player) PreThink(g *Game) {
	for _, proc := range p.Processes {
		proc.PreThink(g)
	}
}

func (p *Player) Think(g *Game) {
	if p.Exile_frames > 0 {
		p.Exile_frames--
		return
	}

	// This will clear out old conditions
	p.Stats.Think()
	var dead []int
	for i, process := range p.Processes {
		process.Think(g)
		if process.Phase() == PhaseComplete {
			dead = append(dead, i)
		}
	}
	for _, i := range dead {
		delete(p.Processes, i)
	}
	// And here we add back in all processes that are still alive.
	for _, process := range p.Processes {
		p.Stats.ApplyCondition(process)
	}

	if p.Delta.Speed > p.Stats.MaxAcc() {
		p.Delta.Speed = p.Stats.MaxAcc()
	}
	if p.Delta.Speed < -p.Stats.MaxAcc() {
		p.Delta.Speed = -p.Stats.MaxAcc()
	}
	if p.Delta.Angle < -p.Stats.MaxTurn() {
		p.Delta.Angle = -p.Stats.MaxTurn()
	}
	if p.Delta.Angle > p.Stats.MaxTurn() {
		p.Delta.Angle = p.Stats.MaxTurn()
	}

	in_lava := false
	for _, lava := range g.Room.Lava {
		if vecInsideConvexPoly(p.Pos(), lava) {
			in_lava = true
		}
	}
	if in_lava {
		p.Stats.ApplyDamage(stats.Damage{stats.DamageFire, 5})
	}

	p.Vx += p.Delta.Speed * math.Cos(p.Angle)
	p.Vy += p.Delta.Speed * math.Sin(p.Angle)
	mangle := math.Atan2(p.Vy, p.Vx)
	friction := g.Friction
	if in_lava {
		friction = g.Friction_lava
	}
	p.Vx *= math.Pow(friction, 1+3*math.Abs(math.Sin(p.Angle-mangle)))
	p.Vy *= math.Pow(friction, 1+3*math.Abs(math.Sin(p.Angle-mangle)))

	// We pretend that the player is started from a little behind wherever they
	// actually are.  This makes it a lot easier to get collisions to make sense
	// from frame to frame.
	epsilon := (linear.Vec2{p.Vx, p.Vy}).Norm().Scale(0.1)
	move := linear.MakeSeg2(p.X-epsilon.X, p.Y-epsilon.Y, p.X+p.Vx, p.Y+p.Vy)
	size := 12.0
	px := p.X
	py := p.Y
	p.X += p.Vx
	p.Y += p.Vy
	for _, poly := range g.Room.Walls {
		for i := range poly {
			// First check against the leading vertex
			{
				v := poly[i]
				dist := v.DistToLine(move)
				if v.Sub(move.Q).Mag() < size {
					dist = v.Sub(move.Q).Mag()
					// Add a little extra here otherwise a player can sneak into geometry
					// through the corners
					ray := move.Q.Sub(v).Norm().Scale(size + 0.1)
					final := v.Add(ray)
					move.Q.X = final.X
					move.Q.Y = final.Y
				} else if dist < size {
					// TODO: This tries to prevent passthrough but has other problems
					// cross := move.Ray().Cross()
					// perp := linear.Seg2{v, cross.Sub(v)}
					// if perp.Left(move.P) != perp.Left(move.Q) {
					//   shift := perp.Ray().Norm().Scale(size - dist)
					//   move.Q.X += shift.X
					//   move.Q.Y += shift.Y
					// }
				}
			}

			// Now check against the segment itself
			w := poly.Seg(i)
			if w.Ray().Cross().Dot(move.Ray()) <= 0 {
				shift := w.Ray().Cross().Norm().Scale(size)
				col := linear.Seg2{shift.Add(w.P), shift.Add(w.Q)}
				if move.DoesIsect(col) {
					cross := col.Ray().Cross()
					fix := linear.Seg2{move.Q, cross.Add(move.Q)}
					isect := fix.Isect(col)
					move.Q.X = isect.X
					move.Q.Y = isect.Y
				}
			}
		}
	}
	p.X = move.Q.X
	p.Y = move.Q.Y
	p.Vx = p.X - px
	p.Vy = p.Y - py

	p.Angle += p.Delta.Angle
	if p.Angle < 0 {
		p.Angle += math.Pi * 2
	}
	if p.Angle > math.Pi*2 {
		p.Angle -= math.Pi * 2
	}

	// Now that we've set our position properly we can do los
	p.Los.Reset(p.Pos())
	for _, poly := range g.Room.Walls {
		for i := range poly {
			p.Los.DrawSeg(poly.Seg(i))
		}
	}

	p.Delta.Angle = 0
	p.Delta.Speed = 0
}

func (p *Player) Supply(supply Mana) Mana {
	for _, process := range p.Processes {
		supply = process.Supply(supply)
	}
	return supply
}

type Ent interface {
	Draw(g *Game)
	Alive() bool
	Exiled() bool
	PreThink(game *Game)
	Think(game *Game)
	ApplyForce(force linear.Vec2)
	ApplyDamage(damage stats.Damage)
	Mass() float64
	SetId(int)
	Id() int
	Pos() linear.Vec2
	SetPos(pos linear.Vec2)
	Vel() linear.Vec2
	SetVel(vel linear.Vec2)
	Supply(mana Mana) Mana
	Copy() Ent
}

type Game struct {
	ManaSource ManaSource

	Room Room

	Rng *cmwc.Cmwc

	// Dimensions of the board
	Dx, Dy int

	Friction      float64
	Friction_lava float64

	// Last Id assigned to an entity
	Next_id int

	Ents []Ent

	GameThinks int
}

func (g *Game) Init() {
	msOptions := ManaSourceOptions{
		NumSeeds:    20,
		NumNodeRows: 60,
		NumNodeCols: 90,

		BoardLeft:   0,
		BoardTop:    0,
		BoardRight:  float64(g.Dx),
		BoardBottom: float64(g.Dy),

		MaxDrainDistance: 120.0,
		MaxDrainRate:     5.0,

		RegenPerFrame:     0.002,
		NodeMagnitude:     100,
		MinNodeBrightness: 20,
		MaxNodeBrightness: 150,
	}
	g.ManaSource.Init(&msOptions, g.Room.Walls, g.Room.Lava)
}

func init() {
	gob.Register(&Game{})
}

type gameResponderWrapper struct {
	g *Game
}

func (grw *gameResponderWrapper) HandleEventGroup(group gin.EventGroup) {
	grw.g.HandleEventGroup(group)
}

func (grw *gameResponderWrapper) Think(int64) {}

func vecInsideConvexPoly(v linear.Vec2, p linear.Poly) bool {
	for i := range p {
		seg := p.Seg(i)
		if seg.Left(v) {
			return false
		}
	}
	return true
}

func invSquareDist(dist_sq float64) float64 {
	return 1.0 / (dist_sq + 1)
}

func getWeights(distance_squares []float64, value_sum float64, transform func(float64) float64) []float64 {
	weights := make([]float64, len(distance_squares))

	weight_sum := 0.0
	for i, dist_sq := range distance_squares {
		if dist_sq >= 0 {
			weights[i] = transform(dist_sq)
		} else {
			weights[i] = 0
		}
		weight_sum += weights[i]
	}

	for i, w := range weights {
		weights[i] = value_sum * w / weight_sum
	}
	return weights
}

func (g *Game) Merge(g2 *Game) {
	frac := 0.0 // i.e. don't merge
	for i := range g.Ents {
		_p1 := g.Ents[i]
		var p1 *Player
		var ok bool
		if p1, ok = _p1.(*Player); !ok {
			continue
		}
		p2, ok := g2.GetEnt(p1.Id()).(*Player)
		if p2 == nil || !ok {
			continue
		}
		p1.X = frac*p2.X + (1-frac)*p1.X
		p1.Y = frac*p2.Y + (1-frac)*p1.Y
		p1.Angle = frac*p2.Angle + (1-frac)*p1.Angle
	}
}

func (g *Game) Copy() interface{} {
	var g2 Game

	g2.ManaSource = g.ManaSource.Copy()

	g2.Room = g.Room

	g2.Rng = g.Rng.Copy()

	g2.Dx = g.Dx
	g2.Dy = g.Dy
	g2.Friction = g.Friction
	g2.Friction_lava = g.Friction_lava
	g2.Next_id = g.Next_id
	g2.GameThinks = g.GameThinks

	g2.Ents = make([]Ent, len(g.Ents))
	g2.Ents = g2.Ents[0:0]
	for _, ent := range g.Ents {
		g2.Ents = append(g2.Ents, ent.Copy())
	}
	return &g2
}

func (g *Game) OverwriteWith(_g2 interface{}) {
	g2 := _g2.(*Game)
	g.ManaSource.OverwriteWith(&g2.ManaSource)
	g.Rng.OverwriteWith(g2.Rng)
	g.Dx = g2.Dx
	g.Dy = g2.Dy
	g.Friction = g2.Friction
	g.Room.Walls = g2.Room.Walls
	g.Next_id = g2.Next_id
	g.GameThinks = g2.GameThinks

	g.Ents = g.Ents[0:0]
	for _, ent := range g2.Ents {
		g.Ents = append(g.Ents, ent.Copy())
	}
}

func (g *Game) GetEnt(id int) Ent {
	for i := range g.Ents {
		if g.Ents[i].Id() == id {
			return g.Ents[i]
		}
	}
	return nil
}

func (g *Game) AddEnt(ent Ent) int {
	g.Next_id++
	ent.SetId(g.Next_id)
	g.Ents = append(g.Ents, ent)
	return g.Ents[len(g.Ents)-1].Id()
}

func (g *Game) ThinkFirst() {}
func (g *Game) ThinkFinal() {}
func (g *Game) Think() {
	g.GameThinks++

	algorithm.Choose(&g.Ents, func(e Ent) bool { return e.Alive() })

	for i := range g.Ents {
		g.Ents[i].PreThink(g)
	}

	g.ManaSource.Think(g.Ents)

	// Advance players, check for collisions, add segments
	for i := range g.Ents {
		if !g.Ents[i].Alive() {
			continue
		}
		g.Ents[i].Think(g)
		pos := g.Ents[i].Pos()
		pos.X = clamp(pos.X, 0, float64(g.Dx))
		pos.Y = clamp(pos.Y, 0, float64(g.Dy))
		g.Ents[i].SetPos(pos)
	}
	moved := make(map[int]bool)
	for i := range g.Ents {
		for j := range g.Ents {
			if i == j || g.Ents[i].Exiled() || g.Ents[j].Exiled() {
				continue
			}
			dist := g.Ents[i].Pos().Sub(g.Ents[j].Pos()).Mag()
			if dist > 25 {
				continue
			}
			if dist < 0.01 {
				continue
			}
			if dist <= 0.5 {
				dist = 0.5
			}
			force := 20.0 * (25 - dist)
			g.Ents[i].ApplyForce(g.Ents[i].Pos().Sub(g.Ents[j].Pos()).Norm().Scale(force))
			moved[i] = true
		}
	}
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

type Turn struct {
	Player_id int
	Delta     float64
}

func init() {
	gob.Register(Turn{})
}

func (t Turn) Apply(_g interface{}) {
	g := _g.(*Game)
	_player := g.GetEnt(t.Player_id)
	if _player == nil {
		return
	}
	player := _player.(*Player)
	player.Delta.Angle = t.Delta
}

type Accelerate struct {
	Player_id int
	Delta     float64
}

func init() {
	gob.Register(Accelerate{})
}

func (a Accelerate) Apply(_g interface{}) {
	g := _g.(*Game)
	_player := g.GetEnt(a.Player_id)
	if _player == nil {
		return
	}
	player := _player.(*Player)
	player.Delta.Speed = a.Delta / 2
}

type GameWindow struct {
	Engine    *cgf.Engine
	game      *Game
	prev_game *Game
	region    gui.Region

	nodeTextureId      gl.Uint
	nodeTextureData    []byte
	nodeWarpingTexture gl.Uint
	nodeWarpingData    []byte
}

func (gw *GameWindow) String() string {
	return "game window"
}
func (gw *GameWindow) Expandable() (bool, bool) {
	return false, false
}
func (gw *GameWindow) Requested() gui.Dims {
	if gw.game == nil {
		return gui.Dims{}
	}
	return gui.Dims{gw.game.Dx, gw.game.Dy}
}
func (gw *GameWindow) Rendered() gui.Region {
	return gw.region
}
func (gw *GameWindow) Think(g *gui.Gui, t int64) {
	if gw.game == nil {
		gw.game = gw.Engine.CopyState().(*Game)
		gw.prev_game = gw.game.Copy().(*Game)
	} else {
		gw.Engine.UpdateState(gw.game)
		gw.game.Merge(gw.prev_game)
		gw.prev_game.OverwriteWith(gw.game)
	}
}
func (gw *GameWindow) Respond(g *gui.Gui, group gui.EventGroup) bool {
	return false
}

var latest_region gui.Region

// Returns the most recent region used when rendering the game.
func (g *Game) Region() gui.Region {
	return latest_region
}

func (gw *GameWindow) Draw(region gui.Region) {
	gw.region = region
	latest_region = region
	gl.PushMatrix()
	defer gl.PopMatrix()
	gl.Translated(gl.Double(gw.region.X), gl.Double(gw.region.Y), 0)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gw.game.ManaSource.Draw(gw, float64(gw.game.Dx), float64(gw.game.Dy))

	gl.Begin(gl.LINES)
	gl.Color4d(1, 1, 1, 1)
	for _, poly := range gw.game.Room.Walls {
		for i := range poly {
			seg := poly.Seg(i)
			gl.Vertex2d(gl.Double(seg.P.X), gl.Double(seg.P.Y))
			gl.Vertex2d(gl.Double(seg.Q.X), gl.Double(seg.Q.Y))
		}
	}
	gl.End()

	gl.Color4d(1, 0, 0, 1)
	for _, poly := range gw.game.Room.Lava {
		gl.Begin(gl.TRIANGLE_FAN)
		for _, v := range poly {
			gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		}
		gl.End()
	}

	gl.Color4d(1, 1, 1, 1)
	losCount := 0
	for _, ent := range gw.game.Ents {
		p, ok := ent.(*Player)
		if !ok {
			continue
		}
		p.Los.WriteDepthBuffer(local.los.texData[losCount], LosMaxDist)
		losCount++
	}
	gw.game.RenderLocal(region)
	gl.Color4d(1, 1, 1, 1)
	for _, ent := range gw.game.Ents {
		ent.Draw(gw.game)
	}
	gl.Disable(gl.TEXTURE_2D)

	for _, player := range local.players {
		if player.active_ability != nil {
			player.active_ability.Draw(player.id, gw.game)
		}
	}
	// base.GetDictionary("luxisr").RenderString("monkeys!!!", 10, 10, 0, float64(gw.game.Game_thinks), gin.Left)
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
