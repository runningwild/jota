uniform float frac;
uniform float inner;
uniform float outer;
uniform float buffer;

void main(void) {
  if (inner < 0.0 || inner > outer || frac < 0.0 || frac > 1.0) {
    gl_FragColor = vec4(1.0, 0.0, 1.0, 1.0);
    return;
  }
  vec2 point = vec2(gl_TexCoord[0].x - 0.5, 0.5 + gl_TexCoord[0].y);
  float angle = atan(point.y, point.x);
  float dist = length(point);
  if (dist < inner - buffer || dist > outer + buffer) {
    gl_FragColor = vec4(0.0, 0.0, 0.0, 0.0);
    return;
  }
  float alpha = 1.0;
  if (dist < inner) {
    alpha = smoothstep(inner - buffer, inner, dist);
  }
  if (dist > outer) {
    alpha = smoothstep(outer + buffer, outer, dist);
  }
  angle = ((angle / 3.1415926535) + 1.0) / 2.0;
  angle += 0.75;
  if (angle > 1.0) {
    angle -= 1.0;
  }
  if (angle > frac) {
    gl_FragColor = vec4(0.0, 0.0, 0.0, 0.0);
    return;
  }
  vec4 c = gl_Color;
  c.a = alpha;
  gl_FragColor = c;
  return;
}

