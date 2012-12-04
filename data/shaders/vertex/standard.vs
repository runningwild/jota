varying vec3 pos;

void main() {
  gl_Position = ftransform();
  gl_ClipVertex = gl_ModelViewMatrix * gl_Vertex;
  gl_FrontColor = gl_Color;
  gl_TexCoord[0] = gl_MultiTexCoord0;
  gl_TexCoord[1] = gl_MultiTexCoord1;
  pos = gl_Vertex.xyz;
}

