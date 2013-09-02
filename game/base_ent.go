package game

import (
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/stats"
	"math"
)

type BaseEnt struct {
	StatsInst stats.Inst
	Position  linear.Vec2
	Velocity  linear.Vec2
	Angle     float64
	Delta     struct {
		Speed float64
		Angle float64
	}
	Gid          Gid
	Side_        int
	CurrentLevel Gid
	// Processes contains all of the processes that this player is casting
	// right now.
	Processes map[int]Process
}

func (b *BaseEnt) Side() int {
	return b.Side_
}
func (b *BaseEnt) OnDeath(g *Game) {
}
func (b *BaseEnt) Walls() [][]linear.Vec2 {
	return nil
}

func (b *BaseEnt) ApplyForce(f linear.Vec2) {
	b.Velocity = b.Velocity.Add(f.Scale(1 / b.Mass()))
}

func (b *BaseEnt) Stats() *stats.Inst {
	return &b.StatsInst
}

func (b *BaseEnt) Mass() float64 {
	return b.StatsInst.Mass()
}

func (b *BaseEnt) Id() Gid {
	return b.Gid
}

func (b *BaseEnt) Pos() linear.Vec2 {
	return b.Position
}

func (b *BaseEnt) Vel() linear.Vec2 {
	return b.Velocity
}

func (b *BaseEnt) SetPos(pos linear.Vec2) {
	b.Position = pos
}

func (b *BaseEnt) Level() Gid {
	return b.CurrentLevel
}

func (b *BaseEnt) SetLevel(level Gid) {
	b.CurrentLevel = level
}

func (b *BaseEnt) Think(g *Game) {
	// This will clear out old conditions
	b.StatsInst.Think()
	var dead []int
	base.DoOrdered(b.Processes, func(a, b int) bool { return a < b }, func(id int, proc Process) {
		proc.Think(g)
		if proc.Phase() == PhaseComplete {
			dead = append(dead, id)
		} else {
			b.StatsInst.ApplyCondition(proc)
		}
	})
	// Removed dead processes from the ent
	for _, id := range dead {
		delete(b.Processes, id)
	}

	if b.Delta.Speed > b.StatsInst.MaxAcc() {
		b.Delta.Speed = b.StatsInst.MaxAcc()
	}
	if b.Delta.Speed < -b.StatsInst.MaxAcc() {
		b.Delta.Speed = -b.StatsInst.MaxAcc()
	}
	if b.Delta.Angle < -b.StatsInst.MaxTurn() {
		b.Delta.Angle = -b.StatsInst.MaxTurn()
	}
	if b.Delta.Angle > b.StatsInst.MaxTurn() {
		b.Delta.Angle = b.StatsInst.MaxTurn()
	}

	room := g.Levels[b.CurrentLevel].Room
	inLava := false
	for _, lava := range room.Lava {
		if linear.VecInsideConvexPoly(b.Pos(), lava) {
			inLava = true
		}
	}
	if inLava {
		b.StatsInst.ApplyDamage(stats.Damage{stats.DamageFire, 5})
	}

	delta := (linear.Vec2{1, 0}).Rotate(b.Angle).Scale(b.Delta.Speed)
	b.Velocity = b.Velocity.Add(delta)
	mangle := math.Atan2(b.Velocity.Y, b.Velocity.X)
	friction := g.Friction
	if inLava {
		friction = g.Friction_lava
	}
	b.Velocity = b.Velocity.Scale(
		math.Pow(friction, 1+3*math.Abs(math.Sin(b.Angle-mangle))))

	// We pretend that the player is started from a little behind wherever they
	// actually are.  This makes it a lot easier to get collisions to make sense
	// from frame to frame.
	var epsilon linear.Vec2
	if b.Velocity.Mag2() > 0 {
		epsilon = b.Velocity.Norm().Scale(0.1)
	}
	move := linear.Seg2{b.Position.Sub(epsilon), b.Position.Add(b.Velocity)}
	size := 12.0
	sizeSq := size * size
	prev := b.Position
	b.Position = b.Position.Add(b.Velocity)
	// walls := g.temp.AllWalls[b.CurrentLevel]
	walls := g.temp.WallCache[b.CurrentLevel].GetWalls(int(b.Position.X), int(b.Position.Y))
	for _, wall := range walls {
		// Don't bother with back-facing segments
		if wall.Right(b.Position) {
			continue
		}
		// First check against the leading vertex
		{
			v := wall.P
			distSq := v.DistSquaredToLine(move)
			if v.Sub(move.Q).Mag2() < sizeSq {
				distSq = v.Sub(move.Q).Mag2()
				// If for some dumb reason an ent is on a vertex this will asplode,
				// so just ignore that case.
				if distSq == 0 {
					continue
				}
				// Add a little extra here otherwise a player can sneak into geometry
				// through the corners
				ray := move.Q.Sub(v).Norm().Scale(size + 0.1)
				final := v.Add(ray)
				move.Q = final
			} else if distSq < sizeSq {
				// TODO: This tries to prevent passthrough but has other problems
				// cross := move.Ray().Cross()
				// perp := linear.Seg2{v, cross.Sub(v)}
				// if perb.Left(move.P) != perb.Left(move.Q) {
				//   shift := perb.Ray().Norm().Scale(size - dist)
				//   move.Q.X += shift.X
				//   move.Q.Y += shift.Y
				// }
			}
		}

		// Now check against the segment itself
		if wall.Ray().Cross().Dot(move.Ray()) <= 0 {
			shift := wall.Ray().Cross().Norm().Scale(size)
			col := linear.Seg2{shift.Add(wall.P), shift.Add(wall.Q)}
			if move.DoesIsect(col) {
				cross := col.Ray().Cross()
				fix := linear.Seg2{move.Q, cross.Add(move.Q)}
				isect := fix.Isect(col)
				move.Q = isect
			}
		}
	}
	b.Position = move.Q
	b.Velocity = b.Position.Sub(prev)

	// b.Velocity.X += float64(g.Rng.Int63()%21-10) / 1000
	// b.Velocity.Y += float64(g.Rng.Int63()%21-10) / 1000

	b.Angle += b.Delta.Angle
	if b.Angle < 0 {
		b.Angle += math.Pi * 2
	}
	if b.Angle > math.Pi*2 {
		b.Angle -= math.Pi * 2
	}

	b.Delta.Angle = 0
	b.Delta.Speed = 0
}
