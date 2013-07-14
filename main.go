package main

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gos"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/system"
	"github.com/runningwild/linear"
	"time"
	// "math"
	"github.com/runningwild/cgf"
	_ "github.com/runningwild/magnus/ability"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
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

func main() {
	fmt.Printf("%v\n", key_map)
	sys.Startup()
	err := gl.Init()
	if err != nil {
		panic(err)
	}

	render.Init()
	render.Queue(func() {
		sys.CreateWindow(10, 10, wdx, wdy)
		sys.EnableVSync(true)
		err := gl.Init()
		if err != nil {
			panic(err)
		}
	})
	base.InitShaders()
	runtime.GOMAXPROCS(2)
	ui, err = gui.Make(gin.In(), gui.Dims{wdx, wdy}, filepath.Join(datadir, "fonts", "skia.ttf"))
	if err != nil {
		panic(err)
	}
	sys.Think()
	for false && len(sys.GetActiveDevices()[gin.DeviceTypeController]) < 2 {
		time.Sleep(time.Millisecond * 100)
		sys.Think()
	}

	var engine *cgf.Engine
	var room game.Room
	err = base.LoadJson(filepath.Join(base.GetDataDir(), "rooms/basic.json"), &room)
	if err != nil {
		panic(err)
	}
	var players []game.Gid
	if Version() == "host" {
		sys.Think()
		var g game.Game
		g.Rng = cmwc.MakeGoodCmwc()
		g.Rng.SeedWithDevRand()
		g.Ents = make(map[game.Gid]game.Ent)
		g.Friction = 0.97
		g.Friction_lava = 0.85
		g.Room = room

		players = append(players, g.AddPlayer(linear.Vec2{500, 300}).Id())
		players = append(players, g.AddPlayer(linear.Vec2{550, 300}).Id())
		// var pest game.Pest
		// err = json.NewDecoder(bytes.NewBuffer([]byte(`
		//     {
		//       "Base": {
		//         "Mass": 100,
		//         "Health": 300
		//       },
		//       "Dynamic": {
		//         "Health": 300
		//       }
		//     }
		//   `))).Decode(&pest.Stats)
		// if err != nil {
		// 	panic(err)
		// }
		// var snare game.Snare
		// err = json.NewDecoder(bytes.NewBuffer([]byte(`
		//     {
		//       "Base": {
		//         "Mass": 1000000000,
		//         "Health": 10
		//       },
		//       "Dynamic": {
		//         "Health": 10
		//       }
		//     }
		//   `))).Decode(&snare.Stats)
		// if err != nil {
		// 	panic(err)
		// }
		// snare.SetPos(linear.Vec2{300, 200})
		// g.Ents = append(g.Ents, &snare)

		g.Init()
		engine, err = cgf.NewHostEngine(&g, 17, "", 1231, base.Log())
		if err != nil {
			panic(err.Error())
		}
		game.SetLocalEngine(engine, sys, false)
	} else if Version() == "client" {
		engine, err = cgf.NewClientEngine(17, "", 1231, base.Log())
		if err != nil {
			base.Log().Printf("Unable to connect: %v", err)
			panic(err.Error())
		}
		game.SetLocalEngine(engine, sys, true)
	} else {
		base.Log().Fatalf("Unable to handle Version() == '%s'", Version())
	}
	if game.IsArchitect() {

	} else {
		d := sys.GetActiveDevices()
		n := 0
		g := engine.CopyState().(*game.Game)
		for _, index := range d[gin.DeviceTypeController] {
			game.SetLocalPlayer(g.Ents[players[n]].(*game.Player), index)
			n++
			if n > len(players) {
				break
			}
		}
		if len(d[gin.DeviceTypeController]) == 0 {
			game.SetLocalPlayer(g.Ents[players[0]].(*game.Player), 0)
		}
	}
	anchor := gui.MakeAnchorBox(gui.Dims{wdx / 2, wdy / 2})
	ui.AddChild(anchor)
	anchor.AddChild(&game.GameWindow{Engine: engine}, gui.Anchor{0.2, 0.2, 0.2, 0.2})
	var v float64
	var profile_output *os.File
	var num_mem_profiles int
	// ui.AddChild(base.MakeConsole())

	base.LoadAllDictionaries()

	ticker := time.Tick(time.Millisecond * 17)
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
		if gin.In().GetKey(gin.AnyKeyP).FramePressCount() > 0 {
			if profile_output == nil {
				profile_output, err = os.Create(filepath.Join(datadir, "cpu.prof"))
				if err == nil {
					err = pprof.StartCPUProfile(profile_output)
					if err != nil {
						fmt.Printf("Unable to start CPU profile: %v\n", err)
						profile_output.Close()
						profile_output = nil
					}
					fmt.Printf("profout: %v\n", profile_output)
				} else {
					fmt.Printf("Unable to start CPU profile: %v\n", err)
				}
			} else {
				pprof.StopCPUProfile()
				profile_output.Close()
				profile_output = nil
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

		v += 0.01
	}
}
