package main

import (
  "encoding/gob"
  "fmt"
  // "runningwild/tron/base"
)

type Ability interface {
  Activate(player *Player, params map[string]int) Process
}

type Process interface {
  // Request returns the most mana that this Process could use right now.
  // Some Processes can operate at any amount of mana, and some will need to
  // get all of their requested mana before they are able to do anything.
  Request() Mana

  // Supplies mana to the Process and returns the unused portion.
  Supply(Mana) Mana

  Think(*Player, *Game)

  Complete() bool
}

// BLINK
// Causes the player to disappear for [frames] frames, where a frame is 16ms.
// Cost 50000 + [frames]^2 blue mana.
func init() {
  gob.Register(blinkAbility{})
  gob.Register(&blinkProcess{})
}

type blinkAbility struct {
}

func (a *blinkAbility) Activate(player *Player, params map[string]int) Process {
  if len(params) != 1 {
    panic(fmt.Sprintf("Blink requires exactly one parameter, not %v", params))
  }
  if _, ok := params["frames"]; !ok {
    panic(fmt.Sprintf("Blink requires [frames] to be specified, not %v", params))
  }
  frames := params["frames"]
  if frames < 0 {
    panic(fmt.Sprintf("Blink requires [frames] > 0, not %d", frames))
  }
  return &blinkProcess{
    Frames:    int32(frames),
    Remaining: Mana{0, 50000 + float64(frames*frames), 0},
  }
}

type blinkProcess struct {
  Frames    int32
  Remaining Mana
}

func (p *blinkProcess) Request() Mana {
  return p.Remaining
}

// Supplies mana to the process.  Any mana that is unused is returned.
func (p *blinkProcess) Supply(supply Mana) Mana {
  for color := range supply {
    if p.Remaining[color] == 0 {
      continue
    }
    if supply[color] == 0 {
      continue
    }
    if supply[color] > p.Remaining[color] {
      supply[color] -= p.Remaining[color]
      p.Remaining[color] = 0
    } else {
      p.Remaining[color] -= supply[color]
      supply[color] = 0
    }
  }
  return supply
}

func (p *blinkProcess) Think(player *Player, game *Game) {
  if p.Remaining.Magnitude() == 0 {
    player.Exile_frames += p.Frames
    p.Frames = 0
  }
}
func (p *blinkProcess) Complete() bool {
  return p.Frames == 0
}

// // BURST
// // All nearby players are pushed radially outward from this one
// // Cost 50000 + [frames]^2 blue mana.
// func init() {
//   gob.Register(blinkAbility{})
//   gob.Register(&blinkProcess{})
// }

// type blinkAbility struct {
// }

// func (a *blinkAbility) Activate(player *Player, params map[string]int) Process {
//   if len(params) != 1 {
//     panic(fmt.Sprintf("Blink requires exactly one parameter, not %v", params))
//   }
//   if _, ok := params["frames"]; !ok {
//     panic(fmt.Sprintf("Blink requires [frames] to be specified, not %v", params))
//   }
//   frames := params["frames"]
//   if frames < 0 {
//     panic(fmt.Sprintf("Blink requires [frames] > 0, not %d", frames))
//   }
//   return &blinkProcess{
//     Frames:    int32(frames),
//     Remaining: Mana{0, 50000 + float64(frames*frames), 0},
//   }
// }

// type blinkProcess struct {
//   Frames    int32
//   Remaining Mana
// }

// func (p *blinkProcess) Request() Mana {
//   return p.Remaining
// }

// // Supplies mana to the process.  Any mana that is unused is returned.
// func (p *blinkProcess) Supply(supply Mana) Mana {
//   for color := range supply {
//     if p.Remaining[color] == 0 {
//       continue
//     }
//     if supply[color] == 0 {
//       continue
//     }
//     if supply[color] > p.Remaining[color] {
//       supply[color] -= p.Remaining[color]
//       p.Remaining[color] = 0
//     } else {
//       p.Remaining[color] -= supply[color]
//       supply[color] = 0
//     }
//   }
//   return supply
// }

// func (p *blinkProcess) Think(player *Player, game *Game) {
//   if p.Remaining.Magnitude() == 0 {
//     player.Exile_frames += p.Frames
//     p.Frames = 0
//   }
// }
// func (p *blinkProcess) Complete() bool {
//   return p.Frames == 0
// }
