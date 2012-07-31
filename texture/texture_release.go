// In release version we don't want to show a bright pink texture when we're
// waiting to load or have an error, so we use a completely transparent
// texture instead.

// +build release
package texture

import (
  "github.com/runningwild/opengl/gl"
  "github.com/runningwild/opengl/glu"
)

var error_texture gl.Texture

func makeErrorTexture() {
  gl.Enable(gl.TEXTURE_2D)
  error_texture = gl.GenTexture()
  error_texture.Bind(gl.TEXTURE_2D)
  gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
  gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
  gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
  gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
  gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
  transparent := []byte{0, 0, 0, 0}
  glu.Build2DMipmaps(gl.TEXTURE_2D, 4, 1, 1, gl.RGBA, transparent)
}
