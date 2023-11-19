@group(0) @binding(0)
var<storage, read_write> output: array<f32>;

@compute @workgroup_size(64)
fn main(
  @builtin(global_invocation_id)
  global_id : vec3u,

  @builtin(workgroup_id)
  workgroup_id : vec3u,

  @builtin(local_invocation_id)
  local_id : vec3u,
) {
  // Avoid accessing the buffer out of bounds
  if (global_id.x >= 300) {// 1024/4) {
    return;
  }

  //output[global_id.x] = f32(global_id.x) * 1000. + f32(local_id.x);
  output[global_id.x] = f32(workgroup_id.x);
}
