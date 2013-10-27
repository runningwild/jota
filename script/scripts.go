package script

import (
	"fmt"
	"github.com/PuerkitoBio/agora/compiler"
	"github.com/PuerkitoBio/agora/runtime"
	"github.com/PuerkitoBio/agora/runtime/stdlib"
	"github.com/runningwild/cgf"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/game"
	"io"
	"math"
	"os"
	"path/filepath"
	// "time"
)

type jotaResolver struct {
	root string
}

func (jr jotaResolver) Resolve(id string) (io.Reader, error) {
	base.Log().Printf("Opening: %s", jr.root)
	return os.Open(filepath.Join(jr.root, id+".agora"))
}

func newJotaResolver() runtime.ModuleResolver {
	return jotaResolver{filepath.Join(base.GetDataDir(), "scripts")}
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

	// The Gid of the player that this Ai is controlling.  This is used to get the
	// entity when needed.
	myGid game.Gid

	// These keep track of the ai's virtual controller
	controller struct {
		angle float64 // [-pi, pi]
		acc   float64 // [-1.0, 1.0]
	}
}

func (jm *JotaModule) Think() {
	ent := jm.engine.GetState().(*game.Game).Ents[jm.myGid]
	if ent == nil {
		return
	}
	base.Log().Printf("Accelerate: %v", jm.myGid)
}

func (jm *JotaModule) ID() string {
	return "jota"
}
func (jm *JotaModule) SetCtx(ctx *runtime.Ctx) {
	jm.ctx = ctx
}

func (jm *JotaModule) newEnt(gid game.Gid) *agoraEnt {
	ob := runtime.NewObject()
	ent := &agoraEnt{
		Object: ob,
		jm:     jm,
		gid:    gid,
	}
	ob.Set(runtime.String("Pos"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.Pos", ent.pos))
	ob.Set(runtime.String("Vel"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.Vel", ent.vel))
	ob.Set(runtime.String("Angle"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.Angle", ent.angle))
	return ent
}

type agoraEnt struct {
	runtime.Object
	jm  *JotaModule
	gid game.Gid
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
		jm.ob.Set(runtime.String("Turn"), runtime.NewNativeFunc(jm.ctx, "jota.Turn", jm.Turn))
	}
	return jm.ob, nil
}

func (jm *JotaModule) Me(vs ...runtime.Val) runtime.Val {
	jm.engine.Pause()
	defer jm.engine.Unpause()
	return jm.newEnt(jm.myGid)
}

func (jm *JotaModule) Move(vs ...runtime.Val) runtime.Val {
	jm.controller.acc = vs[0].Float()
	jm.engine.ApplyEvent(game.Move{jm.myGid, jm.controller.angle, jm.controller.acc})
	return runtime.Nil
}

func (jm *JotaModule) Turn(vs ...runtime.Val) runtime.Val {
	jm.controller.angle = vs[0].Float()
	jm.engine.ApplyEvent(game.Move{jm.myGid, jm.controller.angle, jm.controller.acc})
	return runtime.Nil
}

func (jm *JotaModule) newVec(x, y float64) *agoraVec {
	ob := runtime.NewObject()
	v := &agoraVec{
		Object: ob,
	}
	ob.Set(runtime.String("Length"), runtime.NewNativeFunc(jm.ctx, "jota.Vec.Length", v.length))
	ob.Set(runtime.String("X"), runtime.Number(x))
	ob.Set(runtime.String("Y"), runtime.Number(y))
	return v
}

type agoraVec struct {
	runtime.Object
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

type GameAi struct {
	jm *JotaModule
}

func (ai *GameAi) Start() {
	ctx := runtime.NewCtx(newJotaResolver(), new(compiler.Compiler))
	ctx.RegisterNativeModule(new(stdlib.TimeMod))
	ctx.RegisterNativeModule(&LogModule{})
	ctx.RegisterNativeModule(ai.jm)
	mod, err := ctx.Load("simple")
	if err != nil {
		panic(err)
	}
	go func() {
		_, err := mod.Run()
		base.Error().Printf("Error running script: %v", err)
	}()
}
func (ai *GameAi) Stop()      {}
func (ai *GameAi) Terminate() {}

func init() {
	game.RegisterAiMaker(Maker)
}

func Maker(name string, engine *cgf.Engine, gid game.Gid) game.Ai {
	ai := GameAi{
		jm: &JotaModule{engine: engine, myGid: gid},
	}
	return &ai
}
