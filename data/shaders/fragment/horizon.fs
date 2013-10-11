varying vec3 pos;
uniform vec2 center;
uniform float horizon;

void main(void) {
  float dist = length(center - pos.xy);
  if (dist > horizon) {
    gl_FragColor = vec4(0.0, 0.0, 0.0, 1.0);
    return;
  }
  gl_FragColor = vec4(0.0, 0.0, 0.0, 0.0);
  return;
}

