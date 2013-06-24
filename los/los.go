package los

import (
	"bytes"
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/linear"
	"math"
)

type losInternal struct {
	ZBuffer []float32
	Rays    []linear.Seg2
	Pos     linear.Vec2
	Horizon float32
}

type Los struct {
	in losInternal
}

func Make(size int, Horizon float64) *Los {
	var l Los
	l.in.Horizon = float32(Horizon * Horizon)
	l.in.ZBuffer = make([]float32, size)
	for i := range l.in.ZBuffer {
		l.in.ZBuffer[i] = l.in.Horizon
	}
	l.in.Rays = make([]linear.Seg2, size)
	for i := range l.in.Rays {
		l.in.Rays[i] = linear.Seg2{
			linear.Vec2{0, 0},
			(linear.Vec2{1, 0}).Rotate(2 * math.Pi * (float64(i)/float64(size) - 0.5)),
		}
	}
	return &l
}
func (l *Los) Copy() *Los {
	var l2 Los
	l2.in.ZBuffer = make([]float32, len(l.in.ZBuffer))
	copy(l2.in.ZBuffer, l.in.ZBuffer)
	l2.in.Horizon = l.in.Horizon
	l2.in.Pos = l.in.Pos

	// This doesn't change - so it's ok to just share it.
	l2.in.Rays = l.in.Rays

	return &l2
}
func (l *Los) WriteDepthBuffer(dst []uint32, maxDist float32) {
	for i := range dst {
		dst[i] = uint32(math.Sqrt(float64(l.in.ZBuffer[i])) / float64(maxDist) * (1<<32 - 1))
		// dst[i] = uint32(float32(l.in.ZBuffer[i]) / (maxDist * maxDist) * 4e9)
		// dst[i] = float32(float32(i) / float32(len(dst)))
	}
}
func (l *Los) Reset(Pos linear.Vec2) {
	l.in.Pos = Pos
	for i := range l.in.ZBuffer {
		l.in.ZBuffer[i] = l.in.Horizon
	}
}
func (l *Los) DrawSeg(seg linear.Seg2) {
	seg.P = seg.P.Sub(l.in.Pos)
	seg.Q = seg.Q.Sub(l.in.Pos)
	wrap := len(l.in.ZBuffer)
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
	start := int(((a1 / (2 * math.Pi)) + 0.5) * float64(len(l.in.ZBuffer)))
	end := int(((a2 / (2 * math.Pi)) + 0.5) * float64(len(l.in.ZBuffer)))

	for i := start % wrap; i != end%wrap; i = (i + 1) % wrap {
		dist2 := float32(l.in.Rays[i].Isect(seg).Mag2())
		// base.Log().Printf("%d: %v\n", i, math.Sqrt(dist2))
		// dist = l.in.Rays[i].Isect(seg).Mag2()

		if dist2 < l.in.ZBuffer[i] {
			l.in.ZBuffer[i] = dist2
		}
	}
}

func (l *Los) Render() {
	var v0, v1 linear.Vec2
	gl.Begin(gl.TRIANGLES)
	v1 = (linear.Vec2{-1, 0}).Scale(math.Sqrt(float64(l.in.ZBuffer[0]))).Add(l.in.Pos)
	for i := 1; i <= len(l.in.ZBuffer); i++ {
		dist := math.Sqrt(float64(l.in.ZBuffer[i%len(l.in.ZBuffer)]))
		angle := 2 * math.Pi * (float64(i%len(l.in.ZBuffer))/float64(len(l.in.ZBuffer)) - 0.5)
		if dist <= 0.0 {
			continue
		}
		v0 = v1
		gl.Color4d(gl.Double(1.0-dist/math.Sqrt(float64(l.in.Horizon))), 1.0, 0.0, 1.0)
		v1 = (linear.Vec2{1, 0}).Rotate(angle).Scale(dist).Add(l.in.Pos)
		gl.Vertex2d(gl.Double(l.in.Pos.X), gl.Double(l.in.Pos.Y))
		gl.Vertex2d(gl.Double(v0.X), gl.Double(v0.Y))
		gl.Vertex2d(gl.Double(v1.X), gl.Double(v1.Y))
	}
	gl.End()
}

func (l Los) GobEncode() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(l.in)
	return buf.Bytes(), err
}

func (l *Los) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err := dec.Decode(&l.in)
	return err
}