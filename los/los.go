package los

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/linear"
	"math"
)

type Los struct {
	zBuffer []float32
	rays    []linear.Seg2
	pos     linear.Vec2
	horizon float32
}

func Make(size int, horizon float64) *Los {
	var l Los
	l.horizon = float32(horizon * horizon)
	l.zBuffer = make([]float32, size)
	for i := range l.zBuffer {
		l.zBuffer[i] = l.horizon
	}
	l.rays = make([]linear.Seg2, size)
	for i := range l.rays {
		l.rays[i] = linear.Seg2{
			linear.Vec2{0, 0},
			(linear.Vec2{1, 0}).Rotate(2 * math.Pi * (float64(i)/float64(size) - 0.5)),
		}
	}
	return &l
}
func (l *Los) Copy() *Los {
	var l2 Los
	l2.zBuffer = make([]float32, len(l.zBuffer))
	copy(l2.zBuffer, l.zBuffer)
	l2.horizon = l.horizon
	l2.pos = l.pos

	// This doesn't change - so it's ok to just share it.
	l2.rays = l.rays

	return &l2
}
func (l *Los) WriteDepthBuffer(dst []uint32, maxDist float32) {
	for i := range dst {
		dst[i] = uint32(math.Sqrt(float64(l.zBuffer[i])) / float64(maxDist) * (1<<32 - 1))
		// dst[i] = uint32(float32(l.zBuffer[i]) / (maxDist * maxDist) * 4e9)
		// dst[i] = float32(float32(i) / float32(len(dst)))
	}
}
func (l *Los) Reset(pos linear.Vec2) {
	l.pos = pos
	for i := range l.zBuffer {
		l.zBuffer[i] = l.horizon
	}
}
func (l *Los) DrawSeg(seg linear.Seg2) {
	seg.P = seg.P.Sub(l.pos)
	seg.Q = seg.Q.Sub(l.pos)
	wrap := len(l.zBuffer)
	a1 := math.Atan2(seg.P.Y, seg.P.X)
	a2 := math.Atan2(seg.Q.Y, seg.Q.X)
	if a1 > a2 {
		a1, a2 = a2, a1
		seg.P, seg.Q = seg.Q, seg.P
	}
	if a2-a1 > math.Pi {
		a1, a2 = a2, a1
		seg.P, seg.Q = seg.Q, seg.P
	}
	start := int(((a1 / (2 * math.Pi)) + 0.5) * float64(len(l.zBuffer)))
	end := int(((a2 / (2 * math.Pi)) + 0.5) * float64(len(l.zBuffer)))

	for i := start % wrap; i != end%wrap; i = (i + 1) % wrap {
		dist2 := float32(l.rays[i].Isect(seg).Mag2())
		// base.Log().Printf("%d: %v\n", i, math.Sqrt(dist2))
		// dist = l.rays[i].Isect(seg).Mag2()

		if dist2 < l.zBuffer[i] {
			l.zBuffer[i] = dist2
		}
	}
}

func (l *Los) Render() {
	var v0, v1 linear.Vec2
	gl.Begin(gl.TRIANGLES)
	v1 = (linear.Vec2{-1, 0}).Scale(math.Sqrt(float64(l.zBuffer[0]))).Add(l.pos)
	for i := 1; i <= len(l.zBuffer); i++ {
		dist := math.Sqrt(float64(l.zBuffer[i%len(l.zBuffer)]))
		angle := 2 * math.Pi * (float64(i%len(l.zBuffer))/float64(len(l.zBuffer)) - 0.5)
		if dist <= 0.0 {
			continue
		}
		v0 = v1
		gl.Color4d(gl.Double(1.0-dist/math.Sqrt(float64(l.horizon))), 1.0, 0.0, 1.0)
		v1 = (linear.Vec2{1, 0}).Rotate(angle).Scale(dist).Add(l.pos)
		gl.Vertex2d(gl.Double(l.pos.X), gl.Double(l.pos.Y))
		gl.Vertex2d(gl.Double(v0.X), gl.Double(v0.Y))
		gl.Vertex2d(gl.Double(v1.X), gl.Double(v1.Y))
	}
	gl.End()
}
