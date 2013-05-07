uniform int width;
uniform int height;
uniform sampler2D sampler;
//varying vec3 pos;

void main(void) {
  vec2 pull = vec2(0.3, -0.5);
  vec2 tpos = gl_TexCoord[0].xy;
  vec2 diff = tpos - pull;
  float d = 10.0 * length(diff);
  float d2 = d * d;
  tpos = tpos + diff / (d2 + 1.0);
  float fx = mod(tpos.x * float(width), 1.0);
  float fy = mod(tpos.y * float(height), 1.0);
  float band = 0.1;
  float blur = 0.09;
  float attx;
  float atty;
  if (fx < 0.5) {
    attx = smoothstep(band + blur, band - blur, fx);
  } else {
    attx = smoothstep(1.0 - band - blur, 1.0 - band + blur, fx);
  }
  if (fy < 0.5) {
    atty = smoothstep(band + blur, band - blur, fy);
  } else {
    atty = smoothstep(1.0 - band - blur, 1.0 - band + blur, fy);
  }
  vec4 color = texture2D(sampler, tpos);
  gl_FragColor = color * (1.0 - (1.0 - attx) * (1.0 - atty));
}

