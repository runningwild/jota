package game

import (
	"github.com/runningwild/linear"
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
	Gid Gid
	// Processes contains all of the processes that this player is casting
	// right now.
	Processes map[int]Process
}

func (b *BaseEnt) Copy() *BaseEnt {
	b2 := *b
	b2.Processes = make(map[int]Process)
	for k, v := range b.Processes {
		b2.Processes[k] = v.Copy()
		if v == nil {
			panic("ASDF")
		}
	}
	b2.StatsInst = *b.StatsInst.Copy()
	return &b2
}

func (b *BaseEnt) OnDeath(g *Game) {
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

func (b *BaseEnt) Think(g *Game) {
	// This will clear out old conditions
	b.StatsInst.Think()
	var dead []int
	for i, process := range b.Processes {
		process.Think(g)
		if process.Phase() == PhaseComplete {
			dead = append(dead, i)
		}
	}
	for _, i := range dead {
		delete(b.Processes, i)
	}
	// And here we add back in all processes that are still alive.
	for _, process := range b.Processes {
		b.StatsInst.ApplyCondition(process)
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

	inLava := false
	for _, lava := range g.Room.Lava {
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
	for _, poly := range g.Room.Walls {
		for i := range poly {
			// Don't bother with back-facing segments
			if poly.Seg(i).Right(b.Position) {
				continue
			}
			// First check against the leading vertex
			{
				v := poly[i]
				distSq := v.DistSquaredToLine(move)
				if v.Sub(move.Q).Mag2() < sizeSq {
					distSq = v.Sub(move.Q).Mag2()
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
			w := poly.Seg(i)
			if w.Ray().Cross().Dot(move.Ray()) <= 0 {
				shift := w.Ray().Cross().Norm().Scale(size)
				col := linear.Seg2{shift.Add(w.P), shift.Add(w.Q)}
				if move.DoesIsect(col) {
					cross := col.Ray().Cross()
					fix := linear.Seg2{move.Q, cross.Add(move.Q)}
					isect := fix.Isect(col)
					move.Q = isect
				}
			}
		}
	}
	b.Position = move.Q
	b.Velocity = b.Position.Sub(prev)

	b.Velocity.X += float64(g.Rng.Int63()%21-10) / 1000
	b.Velocity.Y += float64(g.Rng.Int63()%21-10) / 1000

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
