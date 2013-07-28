package gui

import ()

type ButtonList struct {
	ParentResponderWidget
	focusIndex int
}

func (b *ButtonList) Draw(region Region, style StyleStack) {
	region.Dy /= len(b.Children)
	for _, child := range b.Children {
		child.Draw(region, style)
		region.Y += region.Dy
	}
}
func (b *ButtonList) RequestedDims() Dims {
	var dims Dims
	for _, child := range b.Children {
		d := child.RequestedDims()
		if d.Dy > dims.Dy {
			dims.Dy = d.Dy
		}
		if d.Dx > dims.Dx {
			dims.Dx = d.Dx
		}
	}
	dims.Dy *= len(b.Children)
	return dims
}
