package game

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/runningwild/cgf"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/util/algorithm"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/champ"
	"github.com/runningwild/jota/generator"
	"github.com/runningwild/jota/gui"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/linear"
	"math"
	"path/filepath"
	"sync"
	"time"
)

const LosMaxDist = 1000

type EffectMaker func(params map[string]int) Process

var effect_makers map[string]EffectMaker

func RegisterEffect(name string, maker EffectMaker) {
	if effect_makers == nil {
		effect_makers = make(map[string]EffectMaker)
	}
	effect_makers[name] = maker
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
	Think(game *Game)

	// Kills a process.  Any Killed process will return PhaseComplete on any
	// future calls to Phase().
	Kill(game *Game)

	Phase() Phase
}

type Process interface {
	Drain
	Thinker
	stats.Condition
	Draw(source, observer Gid, game *Game)
}

type Color int

const (
	ColorRed Color = iota
	ColorGreen
	ColorBlue
)

var AllColors = []Color{ColorRed, ColorGreen, ColorBlue}

type Ent interface {
	Draw(g *Game)
	Think(game *Game)
	ApplyForce(force linear.Vec2)

	// If this Ent is immovable it may provide walls that will be considered just
	// like normal walls.
	// TODO: Decide whether or not to actually support this
	// Walls() [][]linear.Vec2

	// Stats based methods
	OnDeath(g *Game)
	Stats() *stats.Inst
	Mass() float64
	Vel() linear.Vec2
	Dead() bool

	Id() Gid
	SetId(Gid)
	Pos() linear.Vec2
	Angle() float64
	Side() int // which side the ent belongs to

	// Need to have a SetPos method because we don't want ents moving through
	// walls.
	SetPos(pos linear.Vec2)

	Supply(mana Mana) Mana

	Abilities() []Ability

	BindAi(name string, engine *cgf.Engine)
}

type NonManaUser struct{}

func (NonManaUser) Supply(mana Mana) Mana { return mana }

type Gid string

type Level struct {
	ManaSource ManaSource
	Room       Room
}

type SetupPlayerData struct {
	Side       int
	ChampIndex int
}

type localSetupData struct {
	// Position of the cursor.
	Index int

	// Event handling and engine thinking can happen concurrently, so we need to
	// be able to lock the local data.  Embedded for convenience.
	sync.RWMutex
}

// Used before the 'game' starts to choose sides and characters and whatnot.
type SetupData struct {
	EngineIds []int64                    // engine ids of the engines currently joined
	Players   map[int64]*SetupPlayerData // map from engineid to player data
	Seed      int64                      // random seed

	local localSetupData
}

func (setup *SetupData) HandleEventGroup(group gin.EventGroup, engine *cgf.Engine) {
	if found, event := group.FindEvent(gin.AnyUp); found && event.Type == gin.Press {
		setup.local.Lock()
		defer setup.local.Unlock()
		if setup.local.Index > 0 {
			setup.local.Index--
		}
		return
	}
	if found, event := group.FindEvent(gin.AnyDown); found && event.Type == gin.Press {
		setup.local.Lock()
		defer setup.local.Unlock()
		if setup.local.Index < len(setup.EngineIds) {
			setup.local.Index++
		}
		return
	}

	if found, event := group.FindEvent(gin.AnyLeft); found && event.Type == gin.Press {
		engine.ApplyEvent(SetupChampSelect{engine.Id(), -1})
		return
	}
	if found, event := group.FindEvent(gin.AnyRight); found && event.Type == gin.Press {
		engine.ApplyEvent(SetupChampSelect{engine.Id(), 1})
		return
	}
	if found, event := group.FindEvent(gin.AnyReturn); found && event.Type == gin.Press {
		setup.local.Lock()
		defer setup.local.Unlock()
		if setup.local.Index < len(setup.EngineIds) {
			id := setup.EngineIds[setup.local.Index]
			side := (setup.Players[id].Side + 1) % 2
			engine.ApplyEvent(SetupChangeSides{id, side})
		} else {
			if len(engine.Ids()) > 0 {
				engine.ApplyEvent(SetupComplete{time.Now().UnixNano()})
			}
		}
		return
	}
}

// Because we don't want Think() to be called by both cgf and gin, we put a
// wrapper around Game so that the Think() method called by gin is caught and
// is just a nop.
type GameEventHandleWrapper struct {
	*Game
}

func (GameEventHandleWrapper) Think() {}

func axisControl(v float64) float64 {
	floor := 0.1
	if v < floor {
		return 0.0
	}
	v = (v - floor) / (1.0 - floor)
	v *= v
	return v
}

var control struct {
	any, up, down, left, right gin.Key
}

// Queries the input system for the direction that this controller is moving in
func getControllerDirection(controller gin.DeviceId) linear.Vec2 {
	v := linear.Vec2{
		axisControl(control.right.CurPressAmt()) - axisControl(control.left.CurPressAmt()),
		axisControl(control.down.CurPressAmt()) - axisControl(control.up.CurPressAmt()),
	}
	if v.Mag2() > 1 {
		v = v.Norm()
	}
	return v
}

func (g *Game) HandleEventGroup(group gin.EventGroup) {
	g.local.Engine.Pause()
	defer g.local.Engine.Unpause()
	if g.Setup != nil {
		g.Setup.HandleEventGroup(group, g.local.Engine)
		return
	}

	g.local.RLock()
	defer g.local.RUnlock()

	if found, _ := group.FindEvent(control.any.Id()); found {
		dir := getControllerDirection(gin.DeviceId{gin.DeviceTypeController, gin.DeviceIndexAny})
		g.local.Engine.ApplyEvent(&Move{
			Gid:       g.local.Gid,
			Angle:     dir.Angle(),
			Magnitude: dir.Mag(),
		})
	}

	// ability0Key := gin.In().GetKeyFlat(gin.ControllerButton0+2, gin.DeviceTypeController, gin.DeviceIndexAny)
	// abilityTrigger := gin.In().GetKeyFlat(gin.ControllerButton0+1, gin.DeviceTypeController, gin.DeviceIndexAny)
	ability0Key := gin.In().GetKey(gin.AnyKeyB)
	abilityTrigger := gin.In().GetKey(gin.AnyKeyH)
	foundButton, _ := group.FindEvent(ability0Key.Id())
	foundTrigger, triggerEvent := group.FindEvent(abilityTrigger.Id())
	// TODO: Check if any abilities are Active before sending events to other abilities.
	if foundButton || foundTrigger {
		g.local.Engine.ApplyEvent(UseAbility{
			Gid:     g.local.Gid,
			Index:   0,
			Button:  ability0Key.CurPressAmt(),
			Trigger: foundTrigger && triggerEvent.Type == gin.Press,
		})
	}
}

type SetupSetEngineIds struct {
	EngineIds []int64
}

func (s SetupSetEngineIds) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}
	g.Setup.EngineIds = s.EngineIds
	for _, id := range g.Setup.EngineIds {
		if _, ok := g.Setup.Players[id]; !ok {
			g.Setup.Players[id] = &SetupPlayerData{}
		}
	}
}
func init() {
	gob.Register(SetupSetEngineIds{})
}

type SetupChangeSides struct {
	EngineId int64
	Side     int
}

func (s SetupChangeSides) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}
	g.Setup.Players[s.EngineId].Side = s.Side
}
func init() {
	gob.Register(SetupChangeSides{})
}

type SetupChampSelect struct {
	EngineId int64
	Champ    int
}

func init() {
	gob.Register(SetupChampSelect{})
}
func (s SetupChampSelect) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}
	sideData := g.Setup.Players[s.EngineId]
	if sideData == nil {
		return
	}
	sideData.ChampIndex += s.Champ
	if sideData.ChampIndex < 0 {
		sideData.ChampIndex = 0
	}
	if sideData.ChampIndex >= len(g.Champs) {
		sideData.ChampIndex = len(g.Champs) - 1
	}
}

type SetupComplete struct {
	Seed int64
}

func (u SetupComplete) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}
	sideCount := make(map[int]int)
	// Must have at least two sides
	sideCount[0] = 0
	sideCount[1] = 0
	for _, spd := range g.Setup.Players {
		sideCount[spd.Side]++
	}
	aiCount := 0
	for side, count := range sideCount {
		for count < 2 {
			aiCount++
			count++
			g.Setup.Players[int64(-aiCount)] = &SetupPlayerData{
				Side:       side,
				ChampIndex: 0,
			}
		}
	}
	g.Engines = make(map[int64]*PlayerData)
	for id, player := range g.Setup.Players {
		var gid Gid
		if id < 0 {
			gid = Gid(fmt.Sprintf("Ai:%d", -id))
		} else {
			gid = Gid(fmt.Sprintf("Engine:%d", id))
		}
		g.Engines[id] = &PlayerData{
			PlayerGid:  Gid(gid),
			Side:       player.Side,
			ChampIndex: player.ChampIndex,
		}
	}

	// Now that we have the information we can set up a lot of the local data for
	// this engine's player.
	g.local.Side = g.Engines[g.local.Engine.Id()].Side
	g.local.Gid = g.Engines[g.local.Engine.Id()].PlayerGid
	g.local.Data = g.Engines[g.local.Engine.Id()]

	var room Room
	dx, dy := 1024, 1024
	generated := generator.GenerateRoom(float64(dx), float64(dy), 100, 64, u.Seed)
	data, err := json.Marshal(generated)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	err = json.Unmarshal(data, &room)
	// err = base.LoadJson(filepath.Join(base.GetDataDir(), "rooms/basic.json"), &room)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	g.Level = &Level{}
	g.Level.Room = room
	g.Rng = cmwc.MakeGoodCmwc()
	g.Rng.Seed(12313131)
	g.Ents = make(map[Gid]Ent)
	g.Friction = 0.97
	g.losCache = makeLosCache(g.Level.Room.Dx, g.Level.Room.Dy)
	sides := make(map[int][]int64)
	for id, data := range g.Engines {
		sides[data.Side] = append(sides[data.Side], id)
	}
	var playerDatas []*PlayerData
	for _, playerData := range g.Engines {
		playerDatas = append(playerDatas, playerData)
	}
	for id, ed := range g.Engines {
		base.Log().Printf("%v -> %v", id, *ed)
	}
	g.AddPlayers(playerDatas)

	g.MakeControlPoints()
	g.Init()
	base.Log().Printf("Nillifying g.Setup()")
	g.Setup = nil
}
func init() {
	gob.Register(SetupComplete{})
}

type PlayerData struct {
	// If the player's Ent is in the game then its Gid will match this one.
	PlayerGid Gid

	// If positive, this is the number of frames remaining until the player
	// respawns.
	CountdownFrames int

	Side int

	// Index into the champs array of the champion that this player is using.
	ChampIndex int
}

// All of these values apply to the local player only
type localGameData struct {
	Engine    *cgf.Engine
	Camera    cameraInfo
	Gid       Gid
	Side      int
	Abilities []Ability

	// As long as we're using the keyboard, this is to make sure we track the
	// value of each key properly, so that you don't stop rotating left if you
	// release the right key, for example.
	Up, Down, Left, Right float64

	// This is just a convenience, it points to the PlayerData in
	// Game.Engines[thisEngineId]
	Data *PlayerData

	// Event handling and engine thinking can happen concurrently, so we need to
	// be able to lock the local data.  Embedded for convenience.
	sync.RWMutex
}

type Game struct {
	Setup *SetupData

	Level *Level

	Rng *cmwc.Cmwc

	Friction float64

	// Last Id assigned to anything
	NextIdValue int

	Ents map[Gid]Ent

	// List of data specific to players/computers
	Engines map[int64]*PlayerData

	// All effects that are not tied to a player.
	Processes []Process

	losCache *losCache

	// Champion defs loaded from the data file.  These are set by the host and
	// sent to clients to make debugging and tuning easier.
	Champs []champ.Champion

	local localGameData

	temp struct {
		// This include all room walls for each room, and all walls declared by any
		// ents in that room.
		AllWalls []linear.Seg2
		// It looks silly, but this is...
		// map[LevelId][x/cacheGridSize][y/cacheGridSize] slice of linear.Poly
		// AllWallsGrid  [][][][]linear.Vec2
		AllWallsDirty bool
		WallCache     *wallCache
		// VisibleWallCache is like WallCache but returns all segments visible from
		// a location, rather than all segments that should be checked for
		// collision.
		VisibleWallCache *wallCache

		// List of all ents, in the order that they should be iterated in.
		AllEnts      []Ent
		AllEntsDirty bool
	}
}

func (g *Game) InitializeClientData() {
	// TODO: Do something useful with this.
}

func (g *Game) NextGid() Gid {
	g.NextIdValue++
	return Gid(fmt.Sprintf("%d", g.NextIdValue))
}

func (g *Game) NextId() int {
	g.NextIdValue++
	return g.NextIdValue
}

func (g *Game) SetEngine(engine *cgf.Engine) {
	if control.up == nil {
		// TODO: This is thread-safe, don't worry, but it is dumb.
		controllerUp := gin.In().GetKeyFlat(gin.ControllerAxis0Negative+1, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.up = gin.In().BindDerivedKey("upKey", gin.In().MakeBinding(controllerUp.Id(), nil, nil), gin.In().MakeBinding(gin.AnyKeyW, nil, nil))
		controllerDown := gin.In().GetKeyFlat(gin.ControllerAxis0Positive+1, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.down = gin.In().BindDerivedKey("downKey", gin.In().MakeBinding(controllerDown.Id(), nil, nil), gin.In().MakeBinding(gin.AnyKeyS, nil, nil))
		controllerLeft := gin.In().GetKeyFlat(gin.ControllerAxis0Positive, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.left = gin.In().BindDerivedKey("leftKey", gin.In().MakeBinding(controllerLeft.Id(), nil, nil), gin.In().MakeBinding(gin.AnyKeyA, nil, nil))
		controllerRight := gin.In().GetKeyFlat(gin.ControllerAxis0Negative, gin.DeviceTypeController, gin.DeviceIndexAny)
		control.right = gin.In().BindDerivedKey("rightKey", gin.In().MakeBinding(controllerRight.Id(), nil, nil), gin.In().MakeBinding(gin.AnyKeyD, nil, nil))
		control.any = gin.In().BindDerivedKey(
			"any",
			gin.In().MakeBinding(control.up.Id(), nil, nil),
			gin.In().MakeBinding(control.down.Id(), nil, nil),
			gin.In().MakeBinding(control.left.Id(), nil, nil),
			gin.In().MakeBinding(control.right.Id(), nil, nil))
	}
	// TODO: Unregister this at some point, nub
	gin.In().RegisterEventListener(GameEventHandleWrapper{g})
	g.local.Engine = engine
}

func MakeGame() *Game {
	var g Game
	g.Setup = &SetupData{}
	g.Setup.Players = make(map[int64]*SetupPlayerData)

	// NOTE: Obviously this isn't threadsafe, but I don't intend to be Init()ing
	// multiple game objects at the same time.
	base.RemoveRegistry("champs")
	base.RegisterRegistry("champs", make(map[string]*champ.ChampionDef))
	base.RegisterAllObjectsInDir("champs", filepath.Join(base.GetDataDir(), "champs"), ".json", "json")

	names := base.GetAllNamesInRegistry("champs")
	g.Champs = make([]champ.Champion, len(names))
	for i, name := range names {
		g.Champs[i].Defname = name
		base.GetObject("champs", &g.Champs[i])
	}
	return &g
}

func (g *Game) Init() {
	msOptions := ManaSourceOptions{
		NumSeeds:    20,
		NumNodeRows: g.Level.Room.Dy / 32,
		NumNodeCols: g.Level.Room.Dx / 32,

		BoardLeft:   0,
		BoardTop:    0,
		BoardRight:  float64(g.Level.Room.Dx),
		BoardBottom: float64(g.Level.Room.Dy),

		MaxDrainDistance: 120.0,
		MaxDrainRate:     5.0,

		RegenPerFrame:     0.002,
		NodeMagnitude:     100,
		MinNodeBrightness: 20,
		MaxNodeBrightness: 150,

		Rng: g.Rng,
	}
	g.Level.ManaSource.Init(&msOptions)
	// Values less than this might be used for ability processes, ect...
	g.NextIdValue = 10000
}

func init() {
	gob.Register(&Game{})
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

func lessGids(a, b Gid) bool {
	return a < b
}

func (g *Game) DoForEnts(f func(Gid, Ent)) {
	base.DoOrdered(g.Ents, lessGids, f)
}

func (g *Game) RemoveEnt(gid Gid) {
	delete(g.Ents, gid)
	g.temp.AllEntsDirty = true
}

func (g *Game) AddEnt(ent Ent) {
	if ent.Id() == "" {
		ent.SetId(g.NextGid())
	}
	g.Ents[ent.Id()] = ent
	g.temp.AllEntsDirty = true
}

// Give the gid of a Player ent (ai or engine) this will return that player's
// side, or -1 if such a player does not exist.
func (g *Game) GidToSide(gid Gid) int {
	for _, p := range g.Engines {
		if p.PlayerGid == gid {
			return p.Side
		}
	}
	return -1
}

func (g *Game) ThinkSetup() {
	g.local.Lock()
	defer g.local.Unlock()
	ids := g.local.Engine.Ids()
	if len(ids) > 0 {
		// This is the host engine - so update the list of ids in case it's changed
		g.local.Engine.ApplyEvent(SetupSetEngineIds{ids})
	}

	if len(g.local.Engine.Ids()) == 0 {
		for i, v := range g.Setup.EngineIds {
			if v == g.local.Engine.Id() {
				g.Setup.local.Index = i
			}
		}
	}
}

func (g *Game) ThinkGame() {
	// cache wall data
	if g.temp.AllWalls == nil || g.temp.AllWallsDirty {
		g.temp.AllWalls = nil
		g.temp.WallCache = nil
		g.temp.VisibleWallCache = nil
		var allWalls []linear.Seg2
		base.DoOrdered(g.Level.Room.Walls, func(a, b string) bool { return a < b }, func(_ string, walls linear.Poly) {
			for i := range walls {
				allWalls = append(allWalls, walls.Seg(i))
			}
		})
		g.temp.AllWalls = allWalls
		g.temp.WallCache = &wallCache{}
		g.temp.WallCache.SetWalls(g.Level.Room.Dx, g.Level.Room.Dy, allWalls, 100)
		g.temp.VisibleWallCache = &wallCache{}
		g.temp.VisibleWallCache.SetWalls(g.Level.Room.Dx, g.Level.Room.Dy, allWalls, stats.LosPlayerHorizon)
	}

	// cache ent data
	for _, ent := range g.temp.AllEnts {
		if ent.Dead() {
			if _, ok := ent.(*PlayerEnt); ok {
				var id int64
				_, err := fmt.Sscanf(string(ent.Id()), "Engine:%d", &id)
				if err != nil {
					_, err = fmt.Sscanf(string(ent.Id()), "Ai:%d", &id)
					id = -id // Ai's engine ids are negative
				}
				if err != nil {
					base.Error().Printf("Unable to parse player id '%v'", ent.Id())
				} else {
					if engineData, ok := g.Engines[id]; ok {
						if !ok {
							base.Error().Printf("Unable to find engine %d for player %v", id, ent.Id())
						} else {
							engineData.CountdownFrames = 60 * 10
							base.Log().Printf("%v died, counting down....", *engineData)
						}
					}
				}
			}
			ent.OnDeath(g)
			g.RemoveEnt(ent.Id())
		}
	}

	// Death countdown
	for _, engineData := range g.Engines {
		if engineData.CountdownFrames > 0 {
			engineData.CountdownFrames--
			if engineData.CountdownFrames == 0 {
				base.Log().Printf("Reading %v", *engineData)
				// TODO: It's a bit janky to do it like this, right?
				g.AddPlayers([]*PlayerData{engineData})
			}
		}
	}

	if g.temp.AllEnts == nil || g.temp.AllEntsDirty {
		g.temp.AllEnts = g.temp.AllEnts[0:0]
		g.DoForEnts(func(gid Gid, ent Ent) {
			g.temp.AllEnts = append(g.temp.AllEnts, ent)
		})
		g.temp.AllEntsDirty = false
	}

	for _, proc := range g.Processes {
		proc.Think(g)
	}
	algorithm.Choose(&g.Processes, func(proc Process) bool { return proc.Phase() != PhaseComplete })

	// Advance players, check for collisions, add segments
	for _, ent := range g.temp.AllEnts {
		ent.Think(g)
		pos := ent.Pos()
		eps := 1.0e-3
		pos.X = clamp(pos.X, eps, float64(g.Level.Room.Dx)-eps)
		pos.Y = clamp(pos.Y, eps, float64(g.Level.Room.Dy)-eps)
		ent.SetPos(pos)
	}

	for i := 0; i < len(g.temp.AllEnts); i++ {
		for j := i + 1; j < len(g.temp.AllEnts); j++ {
			outerEnt := g.temp.AllEnts[i]
			innerEnt := g.temp.AllEnts[j]
			distSq := outerEnt.Pos().Sub(innerEnt.Pos()).Mag2()
			colDist := outerEnt.Stats().Size() + innerEnt.Stats().Size()
			if distSq > colDist*colDist {
				continue
			}
			if distSq < 0.0001 {
				continue
			}
			if distSq <= 0.25 {
				distSq = 0.25
			}
			dist := math.Sqrt(distSq)
			force := 50.0 * (colDist - dist)
			outerEnt.ApplyForce(outerEnt.Pos().Sub(innerEnt.Pos()).Scale(force / dist))
			innerEnt.ApplyForce(innerEnt.Pos().Sub(outerEnt.Pos()).Scale(force / dist))
		}
	}

	g.Level.ManaSource.Think(g.Ents)
}

func (g *Game) Think() {
	defer base.StackCatcher()
	switch {
	case g.Setup != nil:
		g.ThinkSetup()
	default:
		g.ThinkGame()
	}
}

// Returns true iff a has los to b, regardless of distance, except that nothing
// can ever have los to something that is beyond stats.LosPlayerHorizon.
func (g *Game) ExistsLos(a, b linear.Vec2) bool {
	vps := g.losCache.Get(int(a.X), int(a.Y), stats.LosPlayerHorizon)
	x := int(b.X / LosGridSize)
	y := int(b.Y / LosGridSize)
	for _, vp := range vps {
		if vp.X == x && vp.Y == y {
			return true
		}
	}
	return false
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

type Move struct {
	Gid       Gid
	Angle     float64
	Magnitude float64
}

func init() {
	gob.Register(Move{})
}

func (m Move) Apply(_g interface{}) {
	g := _g.(*Game)
	player, ok := g.Ents[m.Gid].(*PlayerEnt)
	if !ok {
		return
	}
	if m.Magnitude == 0 {
		player.Target.Angle = player.Angle_
	} else {
		player.Target.Angle = m.Angle
	}
	player.Delta.Speed = m.Magnitude
}

type GameWindow struct {
	Engine *cgf.Engine
	Dims   gui.Dims
	game   *Game
}

func (gw *GameWindow) String() string {
	return "game window"
}
func (gw *GameWindow) Expandable() (bool, bool) {
	return false, false
}
func (gw *GameWindow) Requested() gui.Dims {
	return gui.Dims{800, 600}
}
func (gw *GameWindow) Think(g *gui.Gui) {
	gw.Engine.Pause()
	// gw.Engine.GetState().(*Game)
	gw.Engine.Unpause()
}
func (gw *GameWindow) Respond(group gin.EventGroup) {
}
func (gw *GameWindow) RequestedDims() gui.Dims {
	return gw.Dims
}

func (gw *GameWindow) DrawFocused(region gui.Region) {}
