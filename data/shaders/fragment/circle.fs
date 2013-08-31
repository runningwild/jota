varying vec3 pos;
uniform float edge;

void main(void) {
  float fade = edge / 2.0;
  vec2 ray = pos.xy - vec2(0.5, 0.5);
  float dist = length(ray);
  if (dist < fade) {
    gl_FragColor = gl_Color;
    return;
  }
  if (dist > 0.5) {
    gl_FragColor = vec4(1.0, 0.0, 1.0, 0.0);
    return;
  }
  float alpha = smoothstep(0.5, fade, dist);
  vec4 c = gl_Color;
  c.a = c.a * alpha;
  gl_FragColor = c;
  return;
}

