package script

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/agora/compiler"
	"github.com/PuerkitoBio/agora/runtime"
	"github.com/PuerkitoBio/agora/runtime/stdlib"
	"github.com/runningwild/cgf"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/linear"
	"io"
	"io/ioutil"
	"math"
	"path/filepath"
	"sort"
	"sync"
)

type jotaResolver struct {
	root string

	cacheMutex sync.Mutex
	cache      map[string][]byte
}

func (jr jotaResolver) Resolve(id string) (io.Reader, error) {
	jr.cacheMutex.Lock()
	defer jr.cacheMutex.Unlock()
	if data, ok := jr.cache[id]; ok {
		return bytes.NewReader(data), nil
	}
	base.Log().Printf("Opening: %s", jr.root)
	data, err := ioutil.ReadFile(filepath.Join(jr.root, id+".agora"))
	if err != nil {
		return nil, err
	}
	jr.cache[id] = data
	return bytes.NewReader(data), nil
}

var globalJotaResolverOnce sync.Once
var globalJotaResolver jotaResolver

func getGlobalJotaResolver() *jotaResolver {
	globalJotaResolverOnce.Do(func() {
		globalJotaResolver = jotaResolver{
			cache: make(map[string][]byte),
			root:  filepath.Join(base.GetDataDir(), "scripts"),
		}
	})
	return &globalJotaResolver
}

type LogModule struct {
	// The execution context
	ctx *runtime.Ctx

	// The returned value
	ob runtime.Object
}

func (lm *LogModule) ID() string {
	return "log"
}
func (lm *LogModule) SetCtx(ctx *runtime.Ctx) {
	lm.ctx = ctx
}

// Not interested in any argument in this case. Note the named return values.
func (lm *LogModule) Run(_ ...runtime.Val) (v runtime.Val, err error) {
	// Handle the panics, convert to an error
	defer runtime.PanicToError(&err)
	// Check the cache, create the return value if unavailable
	if lm.ob == nil {
		// Prepare the object
		lm.ob = runtime.NewObject()
		// Export some functions...
		lm.ob.Set(runtime.String("Printf"), runtime.NewNativeFunc(lm.ctx, "log.Printf", lm.Printf))
	}
	return lm.ob, nil
}
func (lm *LogModule) Printf(vs ...runtime.Val) runtime.Val {
	var args []interface{}
	for _, v := range vs[1:] {
		args = append(args, v.Native())
	}
	base.Log().Printf(vs[0].String(), args...)
	return runtime.Nil
}

type JotaModule struct {
	// The execution context
	ctx *runtime.Ctx

	// The returned value
	ob runtime.Object

	// The engine.  The context needs this because all functions that look at any
	// game state will need to pause the engine so the data isn't changed
	// underneath it.
	engine *cgf.Engine

	// Name of the script that this module will load.
	name string

	// The Gid of the player that this Ai is controlling.  This is used to get the
	// entity when needed.
	myGid game.Gid

	terminated bool

	// These keep track of the ai's virtual controller
	controller struct {
		angle float64 // [-pi, pi]
		acc   float64 // [-1.0, 1.0]
	}

	paramsMutex sync.Mutex
	params      map[string]interface{}

	gidToAgoraEntMutex sync.Mutex
	gidToAgoraEnt      map[game.Gid]*agoraEnt
}

func (jm *JotaModule) dieOnTerminated() {
	if jm.terminated {
		panic("module terminated")
	}
}

func (jm *JotaModule) ID() string {
	return "jota"
}
func (jm *JotaModule) SetCtx(ctx *runtime.Ctx) {
	jm.ctx = ctx
}

func (jm *JotaModule) newEnt(gid game.Gid) *agoraEnt {
	jm.gidToAgoraEntMutex.Lock()
	defer jm.gidToAgoraEntMutex.Unlock()
	if _, ok := jm.gidToAgoraEnt[gid]; !ok {
		ob := runtime.NewObject()
		ent := &agoraEnt{
			Object: ob,
			jm:     jm,
			gid:    gid,
		}
		ob.Set(runtime.String("Pos"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.Pos", ent.pos))
		ob.Set(runtime.String("Side"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.Side", ent.side))
		ob.Set(runtime.String("Vel"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.Vel", ent.vel))
		ob.Set(runtime.String("Angle"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.Angle", ent.angle))
		ob.Set(runtime.String("IsPlayer"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.IsPlayer", ent.isType(game.EntTypePlayer)))
		ob.Set(runtime.String("IsCreep"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.IsCreep", ent.isType(game.EntTypeCreep)))
		ob.Set(runtime.String("IsControlPoint"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.IsControlPoint", ent.isType(game.EntTypeControlPoint)))
		ob.Set(runtime.String("IsObstacle"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.IsObstacle", ent.isType(game.EntTypeObstacle)))
		ob.Set(runtime.String("IsProjectile"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.IsProjectile", ent.isType(game.EntTypeProjectile)))
		jm.gidToAgoraEnt[gid] = ent
	}
	return jm.gidToAgoraEnt[gid]
}

type agoraEnt struct {
	runtime.Object
	jm  *JotaModule
	gid game.Gid
}

func (aEnt *agoraEnt) side(args ...runtime.Val) runtime.Val {
	aEnt.jm.engine.Pause()
	defer aEnt.jm.engine.Unpause()
	ent := aEnt.jm.engine.GetState().(*game.Game).Ents[aEnt.gid]
	if ent == nil {
		return runtime.Nil
	}
	return runtime.Number(ent.Side())
}

func (aEnt *agoraEnt) pos(args ...runtime.Val) runtime.Val {
	aEnt.jm.engine.Pause()
	defer aEnt.jm.engine.Unpause()
	ent := aEnt.jm.engine.GetState().(*game.Game).Ents[aEnt.gid]
	if ent == nil {
		return runtime.Nil
	}
	return aEnt.jm.newVec(ent.Pos().X, ent.Pos().Y)
}

func (aEnt *agoraEnt) vel(args ...runtime.Val) runtime.Val {
	aEnt.jm.engine.Pause()
	defer aEnt.jm.engine.Unpause()
	ent := aEnt.jm.engine.GetState().(*game.Game).Ents[aEnt.gid]
	if ent == nil {
		return runtime.Nil
	}
	return aEnt.jm.newVec(ent.Vel().X, ent.Vel().Y)
}

func (aEnt *agoraEnt) angle(args ...runtime.Val) runtime.Val {
	aEnt.jm.engine.Pause()
	defer aEnt.jm.engine.Unpause()
	ent := aEnt.jm.engine.GetState().(*game.Game).Ents[aEnt.gid]
	if ent == nil {
		return runtime.Nil
	}
	return runtime.Number(ent.Angle())
}

func (aEnt *agoraEnt) isType(entType game.EntType) runtime.FuncFn {
	return func(args ...runtime.Val) runtime.Val {
		aEnt.jm.engine.Pause()
		defer aEnt.jm.engine.Unpause()
		ent := aEnt.jm.engine.GetState().(*game.Game).Ents[aEnt.gid]
		if ent == nil {
			return runtime.Bool(false)
		}
		return runtime.Bool(ent.Type() == entType)
	}
}

// Not interested in any argument in this case. Note the named return values.
func (jm *JotaModule) Run(_ ...runtime.Val) (v runtime.Val, err error) {
	// Handle the panics, convert to an error
	defer runtime.PanicToError(&err)
	// Check the cache, create the return value if unavailable
	if jm.ob == nil {
		// Prepare the object
		jm.ob = runtime.NewObject()
		// Export some functions...
		jm.ob.Set(runtime.String("Me"), runtime.NewNativeFunc(jm.ctx, "jota.Me", jm.Me))
		jm.ob.Set(runtime.String("Move"), runtime.NewNativeFunc(jm.ctx, "jota.Move", jm.Move))
		jm.ob.Set(runtime.String("MoveTowards"), runtime.NewNativeFunc(jm.ctx, "jota.MoveTowards", jm.MoveTowards))
		jm.ob.Set(runtime.String("Turn"), runtime.NewNativeFunc(jm.ctx, "jota.Turn", jm.Turn))
		jm.ob.Set(runtime.String("UseAbility"), runtime.NewNativeFunc(jm.ctx, "jota.UseAbility", jm.UseAbility))
		jm.ob.Set(runtime.String("Param"), runtime.NewNativeFunc(jm.ctx, "jota.Param", jm.Param))
		jm.ob.Set(runtime.String("NearestEnt"), runtime.NewNativeFunc(jm.ctx, "jota.NearestEnt", jm.NearestEnt))
		jm.ob.Set(runtime.String("ControlPoints"), runtime.NewNativeFunc(jm.ctx, "jota.ControlPoints", jm.ControlPoints))
		jm.ob.Set(runtime.String("NearbyEnts"), runtime.NewNativeFunc(jm.ctx, "jota.NearbyEnts", jm.NearbyEnts))
		jm.ob.Set(runtime.String("PathDir"), runtime.NewNativeFunc(jm.ctx, "jota.PathDir", jm.PathDir))
	}
	return jm.ob, nil
}

func (jm *JotaModule) Me(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	jm.engine.Pause()
	defer jm.engine.Unpause()
	return jm.newEnt(jm.myGid)
}

func (jm *JotaModule) Move(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	jm.controller.acc = vs[0].Float()
	jm.engine.ApplyEvent(game.Move{jm.myGid, jm.controller.angle, jm.controller.acc})
	return runtime.Nil
}

func (jm *JotaModule) MoveTowards(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	pos, ok := vs[0].Native().(*agoraVec)
	if !ok {
		base.Warn().Printf("Script called MoveTowards with the wrong type: %T", vs[0].Native())
		return runtime.Nil
	}
	jm.engine.Pause()
	defer jm.engine.Unpause()

	g := jm.engine.GetState().(*game.Game)
	me := g.Ents[jm.myGid]
	if me == nil {
		base.Warn().Printf("Darn, I don't exist")
		return runtime.Nil
	}
	angle := pos.Regular().Sub(me.Pos()).Angle()
	jm.engine.ApplyEvent(game.Move{jm.myGid, angle, 1.0})
	return runtime.Nil
}

func (jm *JotaModule) Turn(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	jm.controller.angle = vs[0].Float()
	jm.engine.ApplyEvent(game.Move{jm.myGid, jm.controller.angle, jm.controller.acc})
	return runtime.Nil
}

func (jm *JotaModule) UseAbility(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	jm.engine.ApplyEvent(game.UseAbility{
		Gid:     jm.myGid,
		Index:   int(vs[0].Int()),
		Button:  vs[1].Float(),
		Trigger: vs[2].Bool(),
	})
	return runtime.Nil
}

func (jm *JotaModule) NearestEnt(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	jm.engine.Pause()
	defer jm.engine.Unpause()
	g := jm.engine.GetState().(*game.Game)
	me := g.Ents[jm.myGid]
	if me == nil {
		return runtime.Nil
	}
	var closest game.Ent
	dist := 1e9
	for _, ent := range g.Ents {
		if ent == me {
			continue
		}
		if closest == nil {
			closest = ent
		} else if closest.Pos().Sub(me.Pos()).Mag2() < dist {
			dist = closest.Pos().Sub(me.Pos()).Mag2()
		}
	}
	if closest == nil {
		return runtime.Nil
	}
	return jm.newEnt(closest.Id())
}

type entDistSlice struct {
	ents []game.Ent
	dist []float64
}

func (eds *entDistSlice) Len() int           { return len(eds.ents) }
func (eds *entDistSlice) Less(i, j int) bool { return eds.dist[i] < eds.dist[j] }
func (eds *entDistSlice) Swap(i, j int) {
	eds.ents[i], eds.ents[j] = eds.ents[j], eds.ents[i]
	eds.dist[i], eds.dist[j] = eds.dist[j], eds.dist[i]
}

func (jm *JotaModule) ControlPoints(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	jm.engine.Pause()
	defer jm.engine.Unpause()
	g := jm.engine.GetState().(*game.Game)
	obj := runtime.NewObject()
	count := 0
	for _, ent := range g.Ents {
		if cp, ok := ent.(*game.ControlPoint); ok {
			obj.Set(runtime.Number(count), jm.newEnt(cp.Id()))
			count++
		}
	}
	return obj
}

func (jm *JotaModule) NearbyEnts(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	jm.engine.Pause()
	defer jm.engine.Unpause()
	g := jm.engine.GetState().(*game.Game)
	me := g.Ents[jm.myGid]
	obj := runtime.NewObject()
	if me == nil {
		return obj
	}

	ents := g.EntsInRange(me.Pos(), me.Stats().Vision())
	var eds entDistSlice
	for _, ent := range ents {
		if ent == me {
			continue
		}
		dist := ent.Pos().Sub(me.Pos()).Mag()
		if dist < me.Stats().Vision() && g.ExistsLos(me.Pos(), ent.Pos()) {
			eds.ents = append(eds.ents, ent)
			eds.dist = append(eds.dist, dist)
		}
	}
	sort.Sort(&eds)
	for i, ent := range eds.ents {
		obj.Set(runtime.Number(i), jm.newEnt(ent.Id()))
	}
	return obj
}

func (jm *JotaModule) PathDir(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	jm.engine.Pause()
	defer jm.engine.Unpause()
	g := jm.engine.GetState().(*game.Game)
	pd := g.PathingData()
	src := vs[0].Native().(*agoraVec).Regular()
	dst := vs[1].Native().(*agoraVec).Regular()
	dir := pd.Dir(src, dst)
	return jm.newVec(dir.X, dir.Y)
}

func (jm *JotaModule) Param(vs ...runtime.Val) runtime.Val {
	jm.dieOnTerminated()
	jm.paramsMutex.Lock()
	defer jm.paramsMutex.Unlock()
	paramName := vs[0].String()
	value, ok := jm.params[paramName]
	if !ok {
		return runtime.Nil
	}
	switch t := value.(type) {
	case string:
		return runtime.String(t)
	case bool:
		return runtime.Bool(t)
	case int:
		return runtime.Number(t)
	case float64:
		return runtime.Number(t)
	case linear.Vec2:
		return jm.newVec(t.X, t.Y)
	case game.Gid:
		return jm.newEnt(t)
	default:
		base.Error().Printf("Requested parameter of unexpected type: %T", t)
		return runtime.Nil
	}
}

func (jm *JotaModule) setParam(name string, value interface{}) {
	jm.paramsMutex.Lock()
	defer jm.paramsMutex.Unlock()

	// NOTE: The list of supported types here should match the list in
	// JotaModule.Param()
	switch value.(type) {
	case string:
	case bool:
	case int:
	case float64:
	case linear.Vec2:
	case game.Gid:
	default:
		base.Error().Printf("Tried to specify a parameter with an unexpected type: %T", value)
		return
	}
	jm.params[name] = value
}

func (jm *JotaModule) newVec(x, y float64) *agoraVec {
	ob := runtime.NewObject()
	v := &agoraVec{
		Object: ob,
		jm:     jm,
	}
	ob.Set(runtime.String("Length"), runtime.NewNativeFunc(jm.ctx, "jota.Vec.Length", v.length))
	ob.Set(runtime.String("Sub"), runtime.NewNativeFunc(jm.ctx, "jota.Vec.Sub", v.sub))
	ob.Set(runtime.String("Angle"), runtime.NewNativeFunc(jm.ctx, "jota.Vec.Angle", v.angle))
	ob.Set(runtime.String("X"), runtime.Number(x))
	ob.Set(runtime.String("Y"), runtime.Number(y))
	return v
}

type agoraVec struct {
	runtime.Object
	jm *JotaModule
}

func (v *agoraVec) Regular() linear.Vec2 {
	return linear.Vec2{
		X: v.Get(runtime.String("X")).Float(),
		Y: v.Get(runtime.String("Y")).Float(),
	}
}
func (v *agoraVec) Int() int64          { panic("Bad!") }
func (v *agoraVec) Float() float64      { panic("Bad!") }
func (v *agoraVec) String() string      { return fmt.Sprintf("%v", *v) }
func (v *agoraVec) Bool() bool          { panic("Bad!") }
func (v *agoraVec) Native() interface{} { return v }
func (v *agoraVec) length(args ...runtime.Val) runtime.Val {
	x := v.Get(runtime.String("X")).Float()
	y := v.Get(runtime.String("Y")).Float()
	return runtime.Number(math.Sqrt(x*x + y*y))
}
func (v *agoraVec) sub(args ...runtime.Val) runtime.Val {
	v2 := args[0].Native().(*agoraVec)
	x := v.Get(runtime.String("X")).Float()
	y := v.Get(runtime.String("Y")).Float()
	x2 := v2.Get(runtime.String("X")).Float()
	y2 := v2.Get(runtime.String("Y")).Float()
	return v.jm.newVec(x-x2, y-y2)
}

func (v *agoraVec) angle(args ...runtime.Val) runtime.Val {
	base.Log().Printf("Regul: %v", v.Regular())
	base.Log().Printf("Angle: %v", v.Regular().Angle())
	return runtime.Number(v.Regular().Angle())
}

type GameAi struct {
	jm *JotaModule
}

func (ai *GameAi) Start() {
	if ai.jm == nil {
		return
	}
	ctx := runtime.NewCtx(getGlobalJotaResolver(), new(compiler.Compiler))
	ctx.RegisterNativeModule(new(stdlib.TimeMod))
	ctx.RegisterNativeModule(&LogModule{})
	ctx.RegisterNativeModule(ai.jm)
	mod, err := ctx.Load(ai.jm.name)
	if err != nil {
		base.Error().Printf("Error compiling script: %v", err)
		return
	}
	go func() {
		for {
			_, err := mod.Run()
			if err.Error() == "module terminated" {
				return
			}
			base.Error().Printf("Error running script: %v", err)
		}
	}()
}
func (ai *GameAi) SetParam(name string, value interface{}) {
	if ai.jm == nil {
		return
	}
	ai.jm.setParam(name, value)
}
func (ai *GameAi) Stop() {
	if ai.jm == nil {
		return
	}
}
func (ai *GameAi) Terminate() {
	if ai.jm == nil {
		return
	}
	ai.jm.terminated = true
}

func init() {
	game.RegisterAiMaker(Maker)
}

func Maker(name string, engine *cgf.Engine, gid game.Gid) game.Ai {
	if engine.Ids == nil {
		// Scripts should only run on the host engine
		return &GameAi{}
	}
	ai := GameAi{
		jm: &JotaModule{
			engine:        engine,
			myGid:         gid,
			name:          name,
			params:        make(map[string]interface{}),
			gidToAgoraEnt: make(map[game.Gid]*agoraEnt),
		},
	}
	return &ai
}
