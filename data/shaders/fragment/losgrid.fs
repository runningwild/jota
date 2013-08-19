uniform sampler2D tex1;
uniform sampler2D tex2;
void main() {
  vec4 value1 = texture2D(tex1, gl_TexCoord[0].st);
  vec4 value2 = texture2D(tex2, gl_TexCoord[1].st);
  gl_FragColor = value1 * vec4(value2.w, value2.w, value2.w, 1.0) * gl_Color;
}
