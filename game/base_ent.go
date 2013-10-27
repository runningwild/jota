package game

import (
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/linear"
	"math"
	"sort"
)

type BaseEnt struct {
	StatsInst stats.Inst
	Position  linear.Vec2
	Velocity  linear.Vec2
	Angle_    float64
	Delta     struct {
		Speed float64
		Angle float64
	}
	Target struct {
		Angle float64
	}
	Gid   Gid
	Side_ int
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

func (b *BaseEnt) SetId(gid Gid) {
	b.Gid = gid
}

func (b *BaseEnt) Pos() linear.Vec2 {
	return b.Position
}

func (b *BaseEnt) Vel() linear.Vec2 {
	return b.Velocity
}

func (b *BaseEnt) Angle() float64 {
	return b.Angle_
}

func (b *BaseEnt) SetPos(pos linear.Vec2) {
	b.Position = pos
}

func (b *BaseEnt) Dead() bool {
	return b.Stats().HealthCur() <= 0
}

func (b *BaseEnt) Think(g *Game) {
	// This will clear out old conditions
	b.StatsInst.Think()

	var dead []int
	// Calling DoOrdered is too slow, so we just sort the Gids ourselves and go
	// through them in order.
	pids := make([]int, len(b.Processes))[0:0]
	for pid := range b.Processes {
		pids = append(pids, pid)
	}
	sort.Ints(pids)
	for _, pid := range pids {
		proc := b.Processes[pid]
		proc.Think(g)
		if proc.Phase() == PhaseComplete {
			dead = append(dead, pid)
		} else {
			b.StatsInst.ApplyCondition(proc)
		}
	}

	// Removed dead processes from the ent
	for _, id := range dead {
		delete(b.Processes, id)
	}

	if b.Delta.Speed < -1.0 {
		b.Delta.Speed = -1.0
	}
	if b.Delta.Speed > 1.0 {
		b.Delta.Speed = 1.0
	}

	// TODO: Speed is a complete misnomer now - fix it!
	force := b.Delta.Speed * (linear.Vec2{1, 0}).Rotate(b.Target.Angle).Dot((linear.Vec2{1, 0}).Rotate(b.Angle_))
	b.ApplyForce((linear.Vec2{1, 0}).Rotate(b.Angle_).Scale(force * b.Stats().MaxAcc()))

	mangle := math.Atan2(b.Velocity.Y, b.Velocity.X)
	friction := g.Friction
	b.Velocity = b.Velocity.Scale(
		math.Pow(friction, 1+3*math.Abs(math.Sin(b.Angle_-mangle))))

	if b.Velocity.Mag2() < 0.01 {
		b.Velocity = linear.Vec2{0, 0}
	} else {
		size := b.Stats().Size()
		sizeSq := size * size
		// We pretend that the player is started from a little behind wherever they
		// actually are.  This makes it a lot easier to get collisions to make sense
		// from frame to frame.
		epsilon := b.Velocity.Norm().Scale(size / 2)
		move := linear.Seg2{b.Position.Sub(epsilon), b.Position.Add(b.Velocity)}
		prev := b.Position
		walls := g.temp.WallCache.GetWalls(int(b.Position.X), int(b.Position.Y))
		for _, wall := range walls {
			// Don't bother with back-facing segments
			if wall.Right(b.Position) {
				continue
			}

			// Check against the segment itself
			if wall.Ray().Cross().Dot(move.Ray()) <= 0 {
				shiftNorm := wall.Ray().Cross().Norm()
				shift := shiftNorm.Scale(size)
				col := linear.Seg2{shift.Add(wall.P), shift.Add(wall.Q)}
				if move.DoesIsect(col) {
					cross := col.Ray().Cross()
					fix := linear.Seg2{move.Q, cross.Add(move.Q)}
					isect := fix.Isect(col)
					move.Q = isect
				}
			}
		}
		for _, wall := range walls {
			// Check against the leading vertex
			{
				v := wall.P
				originMove := linear.Seg2{move.P.Sub(v), move.Q.Sub(v)}
				originPerp := linear.Seg2{linear.Vec2{}, move.Ray().Cross()}
				dist := originMove.DistFromOrigin()
				if originPerp.DoesIsect(originMove) && dist < size {
					// Stop passthrough
					isect := originMove.Isect(originPerp).Add(v)
					diff := math.Sqrt(sizeSq - dist*dist)
					finalLength := isect.Sub(move.P).Mag() - diff
					move.Q = move.Ray().Norm().Scale(finalLength).Add(move.P)
				} else if v.Sub(move.Q).Mag2() < sizeSq {
					move.Q = move.Q.Sub(v).Norm().Scale(size).Add(v)
				}
			}
		}
		b.Position = move.Q
		b.Velocity = b.Position.Sub(prev)
	}

	if math.Abs(b.Angle_+b.Target.Angle-math.Pi) < 0.01 {
		b.Angle_ += 0.1
	} else {
		frac := 0.80
		curDir := (linear.Vec2{1, 0}).Rotate(b.Angle_).Scale(frac)
		targetDir := (linear.Vec2{1, 0}).Rotate(b.Target.Angle).Scale(1.0 - frac)
		newDir := curDir.Add(targetDir)
		if newDir.Mag() > 0.01 {
			b.Angle_ = newDir.Angle()
		}
	}
}
