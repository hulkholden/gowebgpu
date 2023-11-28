struct Particles {
  particles : array<Particle>,
}
@binding(0) @group(0) var<uniform> params : SimParams;
@binding(1) @group(0) var<storage, read> particlesA : Particles;
@binding(2) @group(0) var<storage, read_write> particlesB : Particles;

// https://github.com/austinEng/Project6-Vulkan-Flocking/blob/master/data/shaders/computeparticles/particle.comp
@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  var index = GlobalInvocationID.x;

  var vPos = particlesA.particles[index].pos;
  var vVel = particlesA.particles[index].vel;
  var cMass = vec2(0.0);
  var cVel = vec2(0.0);
  var colVel = vec2(0.0);
  var cMassCount = 0u;
  var cVelCount = 0u;

  for (var i = 0u; i < arrayLength(&particlesA.particles); i++) {
    if (i == index) {
      continue;
    }

    let pos = particlesA.particles[i].pos.xy;
    let vel = particlesA.particles[i].vel.xy;
    let dPos = pos - vPos;
    let dist = length(dPos);
    if (dist < params.rule1Distance) {
      cMass += pos;
      cMassCount++;
    }
    if (dist < params.rule2Distance) {
      colVel -= dPos;
    }
    if (dist < params.rule3Distance) {
      cVel += vel;
      cVelCount++;
    }
  }
  if (cMassCount > 0) {
    cMass = (cMass / vec2(f32(cMassCount))) - vPos;
  }
  if (cVelCount > 0) {
    cVel /= f32(cVelCount);
  }
  vVel += (cMass * params.rule1Scale) + (colVel * params.rule2Scale) + (cVel * params.rule3Scale);

  // clamp velocity for a more pleasing simulation
  vVel = normalize(vVel) * clamp(length(vVel), 0.0, 0.1);
  // kinematic update
  vPos = vPos + (vVel * params.deltaT);
  // Wrap around boundary
  if (vPos.x < -1.0) {
    vPos.x += 2.0;
  }
  if (vPos.x > 1.0) {
    vPos.x -= -2.0;
  }
  if (vPos.y < -1.0) {
    vPos.y += 2.0;
  }
  if (vPos.y > 1.0) {
    vPos.y -= -2.0;
  }

  // Write back
  particlesB.particles[index].pos = vPos;
  particlesB.particles[index].vel = vVel;
}
