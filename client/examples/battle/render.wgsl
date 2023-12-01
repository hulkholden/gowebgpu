struct VertexOutput {
  @builtin(position) position : vec4<f32>,
  @location(4) color : vec4<f32>,
}

// TODO: provide this as a matrix.
const worldScale = 1000.0;

@vertex
fn vertex_main(
  @location(0) a_particlePos : vec2<f32>,
  @location(1) a_particleAngle: f32,
  @location(2) a_particleCol : u32,
  @location(3) a_pos : vec2<f32>,
) -> VertexOutput {
  let c = cos(a_particleAngle);
  let s = sin(a_particleAngle);
  let transform = mat2x2f(vec2f(c, -s), vec2f(s, c));
  let pos = (a_particlePos + (a_pos * transform)) / worldScale;

  var output : VertexOutput;
  output.position = vec4(pos, 0.0, 1.0);
  // TODO: why doesn't unpack4xU8 work?
  output.color = vec4(
    f32((a_particleCol >> 16) & 0xff) / 255.0,
    f32((a_particleCol >> 8) & 0xff) / 255.0,
    f32((a_particleCol >> 0) & 0xff) / 255.0,
     1.0);
  return output;
}

@fragment
fn fragment_main(@location(4) color : vec4<f32>) -> @location(0) vec4<f32> {
  return color;
}
