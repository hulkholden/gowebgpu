struct VertexOutput {
  @builtin(position) position : vec4<f32>,
  @location(4) color : vec4<f32>,
}

@vertex
fn vertex_main(
  @location(0) a_particlePos : vec2<f32>,
  @location(1) a_particleVel : vec2<f32>,
  @location(2) a_pos : vec2<f32>
) -> VertexOutput {
  let angle = -atan2(a_particleVel.x, a_particleVel.y);
  let pos = vec2(
    (a_pos.x * cos(angle)) - (a_pos.y * sin(angle)),
    (a_pos.x * sin(angle)) + (a_pos.y * cos(angle))
  );
  let pi = 3.14159265359;

  var output : VertexOutput;
  output.position = vec4(pos + a_particlePos, 0.0, 1.0);
  var rgb = hsl2rgb(vec3((angle / pi) * 0.5 + 0.5, 1.0, 0.5));
  output.color = vec4(rgb, 1.0);
  return output;
}

fn hsl2rgb(c : vec3<f32>) -> vec3<f32> {
  let x = c.x * 6.0 + vec3(0.0,4.0,2.0);
  let y = 6.0;
  let m = x - y * floor(x/y); // mod(x, y);
  let raw = abs(m - 3.0) - 1.0;
  let rgb = clamp(raw, vec3<f32>(0.0), vec3<f32>(1.0));
  return c.z + c.y * (rgb - 0.5) * (1.0 - abs(2.0 * c.z - 1.0));
}

@fragment
fn fragment_main(@location(4) color : vec4<f32>) -> @location(0) vec4<f32> {
  return color;
}
