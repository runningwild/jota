package game

import (
	"fmt"
	"github.com/PuerkitoBio/agora/compiler"
	"github.com/PuerkitoBio/agora/runtime"
	"github.com/PuerkitoBio/agora/runtime/stdlib"
	"github.com/runningwild/cgf"
	"github.com/runningwild/jota/base"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"
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
	myGid Gid
}

func (jm *JotaModule) ID() string {
	return "jota"
}
func (jm *JotaModule) SetCtx(ctx *runtime.Ctx) {
	jm.ctx = ctx
}

func (jm *JotaModule) newEnt(gid Gid) *agoraEnt {
	ob := runtime.NewObject()
	ent := &agoraEnt{
		Object: ob,
		jm:     jm,
		gid:    gid,
	}
	ob.Set(runtime.String("Pos"), runtime.NewNativeFunc(jm.ctx, "jota.Ent.Pos", ent.pos))
	return ent
}

type agoraEnt struct {
	runtime.Object
	jm  *JotaModule
	gid Gid
}

func (aEnt *agoraEnt) pos(args ...runtime.Val) runtime.Val {
	aEnt.jm.engine.Pause()
	defer aEnt.jm.engine.Unpause()
	ent := aEnt.jm.engine.GetState().(*Game).Ents[aEnt.gid]
	if ent == nil {
		return runtime.Nil
	}
	return aEnt.jm.newVec(ent.Pos().X, ent.Pos().Y)
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
		jm.ob.Set(runtime.String("Force"), runtime.NewNativeFunc(jm.ctx, "jota.Force", jm.Force))
	}
	return jm.ob, nil
}

func (jm *JotaModule) Me(vs ...runtime.Val) runtime.Val {
	jm.engine.Pause()
	defer jm.engine.Unpause()
	return jm.newEnt(jm.myGid)
}

func (jm *JotaModule) Force(vs ...runtime.Val) runtime.Val {
	jm.engine.Pause()
	defer jm.engine.Unpause()
	time.Sleep(time.Millisecond * 10)
	ent := jm.engine.GetState().(*Game).Ents[jm.myGid]
	if ent == nil {
		return runtime.Nil
	}
	jm.engine.ApplyEvent(Accelerate{jm.myGid, 10000})
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

func Start(engine *cgf.Engine, gid Gid) {
	ctx := runtime.NewCtx(newJotaResolver(), new(compiler.Compiler))
	ctx.RegisterNativeModule(new(stdlib.TimeMod))
	ctx.RegisterNativeModule(&LogModule{})
	ctx.RegisterNativeModule(&JotaModule{engine: engine, myGid: gid})
	mod, err := ctx.Load("simple")
	if err != nil {
		panic(err)
	}
	base.Log().Printf("Starting!")
	go func() {
		_, err := mod.Run()
		base.Error().Printf("Error running script: %v", err)
	}()
}
