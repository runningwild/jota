package main

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gos"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/system"
	g2 "github.com/runningwild/jota/gui"
	"time"
	// "math"
	"github.com/runningwild/cgf"
	_ "github.com/runningwild/jota/ability"
	_ "github.com/runningwild/jota/ability/kassadin"
	"github.com/runningwild/jota/base"
	_ "github.com/runningwild/jota/effects"
	"github.com/runningwild/jota/game"
	_ "github.com/runningwild/jota/script"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

var (
	sys      system.System
	datadir  string
	ui       *gui.Gui
	wdx, wdy int
	key_map  base.KeyMap
)

func init() {
	runtime.LockOSThread()
	sys = system.Make(gos.GetSystemInterface())

	datadir = filepath.Join(os.Args[0], "..", "..")
	base.SetDatadir(datadir)
	base.Log().Printf("Setting datadir: %s", datadir)
	wdx = 1024
	wdy = 768
	var key_binds base.KeyBinds
	base.LoadJson(filepath.Join(datadir, "key_binds.json"), &key_binds)
	fmt.Printf("Prething: %v\n", key_binds)
	key_map = key_binds.MakeKeyMap()
	base.SetDefaultKeyMap(key_map)
}

func debugHookup(version string) *cgf.Engine {
	var err error
	for false && len(sys.GetActiveDevices()[gin.DeviceTypeController]) < 2 {
		time.Sleep(time.Millisecond * 100)
		sys.Think()
	}

	var engine *cgf.Engine
	if version != "host" {
		res, err := cgf.SearchLANForHosts(20007, 20002, 500)
		if err != nil || len(res) == 0 {
			base.Log().Printf("Unable to connect: %v", err)
			base.Error().Fatalf("%v", err.Error())
		}
		engine, err = cgf.NewClientEngine(17, res[0].Ip, 20007, base.EmailCrashReport, base.Log())
		if err != nil {
			base.Log().Printf("Unable to connect: %v", err)
			base.Error().Fatalf("%v", err.Error())
		}
		engine.GetState().(*game.Game).SetEngine(engine)
	} else {
		sys.Think()
		g := game.MakeGame()
		if version == "host" {
			engine, err = cgf.NewHostEngine(g, 17, "", 20007, base.EmailCrashReport, base.Log())
			if err != nil {
				panic(err)
			}
			err = cgf.Host(20007, "thunderball")
			if err != nil {
				panic(err)
			}
		} else {
			engine, err = cgf.NewLocalEngine(g, 17, base.EmailCrashReport, base.Log())
		}
		if err != nil {
			base.Error().Fatalf("%v", err.Error())
		}
	}

	base.Log().Printf("Engine Id: %v", engine.Id())
	base.Log().Printf("All Ids: %v", engine.Ids())
	return engine
}

func mainLoop(engine *cgf.Engine, mode string) {
	defer engine.Kill()
	var profile_output *os.File
	var contention_output *os.File
	var num_mem_profiles int
	// ui.AddChild(base.MakeConsole())

	ticker := time.Tick(time.Millisecond * 17)
	ui := g2.Make(0, 0, wdx, wdy)
	ui.AddChild(&game.GameWindow{Engine: engine, Dims: g2.Dims{wdx, wdy}}, g2.AnchorDeadCenter)
	ui.AddChild(g2.MakeConsole(wdx, wdy), g2.AnchorDeadCenter)
	// side0Index := gin.In().BindDerivedKeyFamily("Side0", gin.In().MakeBindingFamily(gin.Key1, []gin.KeyIndex{gin.EitherControl}, []bool{true}))
	// side1Index := gin.In().BindDerivedKeyFamily("Side1", gin.In().MakeBindingFamily(gin.Key2, []gin.KeyIndex{gin.EitherControl}, []bool{true}))
	// side2Index := gin.In().BindDerivedKeyFamily("Side2", gin.In().MakeBindingFamily(gin.Key3, []gin.KeyIndex{gin.EitherControl}, []bool{true}))
	// side0Key := gin.In().GetKeyFlat(side0Index, gin.DeviceTypeAny, gin.DeviceIndexAny)
	// side1Key := gin.In().GetKeyFlat(side1Index, gin.DeviceTypeAny, gin.DeviceIndexAny)
	// side2Key := gin.In().GetKeyFlat(side2Index, gin.DeviceTypeAny, gin.DeviceIndexAny)
	defer ui.StopEventListening()
	for {
		<-ticker
		if gin.In().GetKey(gin.AnyEscape).FramePressCount() != 0 {
			return
		}
		sys.Think()
		render.Queue(func() {
			ui.Draw()
		})
		render.Queue(func() {
			sys.SwapBuffers()
		})
		render.Purge()
		// TODO: Replace the 'P' key with an appropriate keybind
		var err error
		if gin.In().GetKey(gin.AnyKeyP).FramePressCount() > 0 {
			if profile_output == nil {
				profile_output, err = os.Create(filepath.Join(datadir, "cpu.prof"))
				if err == nil {
					err = pprof.StartCPUProfile(profile_output)
					if err != nil {
						base.Log().Printf("Unable to start CPU profile: %v\n", err)
						profile_output.Close()
						profile_output = nil
					}
					base.Log().Printf("cpu prof: %v\n", profile_output)
				} else {
					base.Log().Printf("Unable to start CPU profile: %v\n", err)
				}
			} else {
				pprof.StopCPUProfile()
				profile_output.Close()
				profile_output = nil
			}
		}

		if gin.In().GetKey(gin.AnyKeyL).FramePressCount() > 0 {
			if contention_output == nil {
				contention_output, err = os.Create(filepath.Join(datadir, "contention.prof"))
				if err == nil {
					runtime.SetBlockProfileRate(1)
					base.Log().Printf("contention prof: %v\n", contention_output)
				} else {
					base.Log().Printf("Unable to start contention profile: %v\n", err)
				}
			} else {
				pprof.Lookup("block").WriteTo(contention_output, 0)
				contention_output.Close()
				contention_output = nil
			}
		}

		// TODO: Replace the 'M' key with an appropriate keybind
		if gin.In().GetKey(gin.AnyKeyM).FramePressCount() > 0 {
			f, err := os.Create(filepath.Join(datadir, fmt.Sprintf("mem.%d.prof", num_mem_profiles)))
			if err != nil {
				base.Error().Printf("Unable to write mem profile: %v", err)
			}
			pprof.WriteHeapProfile(f)
			f.Close()
			num_mem_profiles++
		}
	}
}

func main() {
	defer base.StackCatcher()
	fmt.Printf("sys.Startup()...")
	sys.Startup()
	fmt.Printf("successful.\n")
	fmt.Printf("gl.Init()...")
	err := gl.Init()
	fmt.Printf("successful.\n")
	if err != nil {
		base.Error().Fatalf("%v", err)
	}

	fmt.Printf("render.Init()...")
	render.Init()
	fmt.Printf("successful.\n")
	render.Queue(func() {
		fmt.Printf("sys.CreateWindow()...")
		sys.CreateWindow(10, 10, wdx, wdy)
		fmt.Printf("successful.\n")
		sys.EnableVSync(true)
	})
	base.InitShaders()
	runtime.GOMAXPROCS(10)
	fmt.Printf("sys.Think()...")
	sys.Think()
	fmt.Printf("successful.\n")

	base.LoadAllDictionaries()

	if Version() != "standard" {
		engine := debugHookup(Version())
		mainLoop(engine, "standard")
	} else {
		// TODO: Reimplement standard hookup
	}
}
