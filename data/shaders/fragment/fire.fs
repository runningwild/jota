varying vec3 pos;
uniform float inner;
uniform float outer;
uniform float frac;
uniform float heading;

void main(void) {
  if (inner < 0.0 || inner > outer || frac < 0.0) {
    gl_FragColor = vec4(1.0, 0.0, 1.0, 1.0);
    return;
  }
  vec2 ray = pos.xy - vec2(0.5, 0.5);
  float angle = atan(ray.y, ray.x) - heading - 3.1415926535/2.0 + 2.0*3.1415926535;
  float dist = length(ray);
  float buffer = 0.002;
  if (dist < inner) {
    gl_FragColor = vec4(0.0, 1.0, 0.0, 0.0);
    return;
  }
  if (dist > outer) {
    gl_FragColor = vec4(0.0, 0.0, 1.0, 0.0);
    return;
  }
  float alpha = 1.0;
  if (dist < inner + buffer) {
    alpha = smoothstep(inner, inner + buffer, dist);
  }
  if (dist > outer - buffer) {
    alpha = smoothstep(outer, outer - buffer, dist);
  }
  angle = ((angle / 3.1415926535) + 1.0) / 2.0;
  angle += 0.75;
  if (angle > 1.0) {
    angle -= 1.0;
  }
  if (angle > 1.0) {
    angle -= 1.0;
  }
  if (angle > frac / 2.0 && angle < 1.0 -frac / 2.0) {
    gl_FragColor = vec4(0.0, 0.0, 0.0, 0.0);
    return;
  }
  vec4 c = gl_Color;
  c.a = c.a * alpha;
  gl_FragColor = c;
  return;
}

