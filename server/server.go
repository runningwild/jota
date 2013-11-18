package main

import (
	"fmt"
	"github.com/runningwild/cgf"
	_ "github.com/runningwild/jota/ability"
	_ "github.com/runningwild/jota/ability/control_point"
	_ "github.com/runningwild/jota/ability/creep"
	"github.com/runningwild/jota/base"
	_ "github.com/runningwild/jota/effects"
	"github.com/runningwild/jota/game"
	_ "github.com/runningwild/jota/script"
)

func main() {
	base.SetDatadir("../data")
	g := game.MakeGame()
	engine, err := cgf.NewHostEngine(g, 17, "", 20007, nil, nil)
	if err != nil {
		fmt.Printf("Unable to create engine: %v\n", err)
		return
	}
	err = cgf.Host(20007, "thunderball")
	if err != nil {
		fmt.Printf("Unable to host: %v\n", err)
		return
	}
	fmt.Printf("%v\n", engine)
	select {}
}
