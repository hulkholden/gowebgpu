struct Particles {
  particles : array<Particle>,
}
@binding(0) @group(0) var<uniform> params : SimParams;
@binding(1) @group(0) var<storage, read> particlesA : Particles;
@binding(2) @group(0) var<storage, read_write> particlesB : Particles;

const pi = 3.14159265359;
const twoPi = 2 * pi;

const proNavGain = 3.0;

const bodyTypeNone = 0u;
const bodyTypeShip = 1u;
const bodyTypeMissile = 2u;

struct Control {
  linearAcc : vec2f,
  angularAcc : f32,
}

struct ReferenceFrame {
  pos : vec2f,
  vel : vec2f,

  angle : f32,
  angularVel : f32,
}

fn referenceFrameSub(a : ReferenceFrame, b : ReferenceFrame) -> ReferenceFrame {
  return ReferenceFrame(a.pos - b.pos, a.vel - b.vel, angleDiff(a.angle, b.angle), a.angularVel - b.angularVel);
}

fn referenceFrameRotate(a : ReferenceFrame, angle : f32) -> ReferenceFrame {
  let c = cos(angle);
  let s = sin(angle);
  let transform = mat2x2f(vec2f(c, -s), vec2f(s, c));
  return ReferenceFrame(a.pos * transform, a.vel * transform, a.angle + angle, a.angularVel);
}

fn rotVec(v : vec2f, a : f32) -> vec2f {
  let c = cos(a);
  let s = sin(a);
  let transform = mat2x2f(vec2f(c, -s), vec2f(s, c));
  return v * transform;
}

fn angleOf(v : vec2f, def : f32) -> f32 {
  return select(def, atan2(-v.x, v.y), length(v) > 0);
}

fn angleDiff(a : f32, b : f32) -> f32 {
  return normalizeAngle(a - b);
}

fn normalizeAngle(a : f32) -> f32 {
  var n = modReplacement(a + pi, twoPi);
  if (n < 0) {
    n += twoPi;
  }
  return n - pi;
}

// modReplacement returns the floating point remainder of x/y.
// TOOD: check whathappens if x is < 0.
fn modReplacement(x : f32, y : f32) -> f32 {
  return x - (y * floor(x/y));
}

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) GlobalInvocationID : vec3<u32>) {
  let index = GlobalInvocationID.x;
  let particle = particlesA.particles[index];

  var f = particleReferenceFrame(particle);

  switch particleType(particle) {
    case bodyTypeNone: {
    }
    case bodyTypeShip: {
      f.vel = flock(particle, index);
      f.angle = angleOf(f.vel, f.angle);
    }
    case bodyTypeMissile: {
      let control = updateMissile(f, index, particle.targetIdx);
      f.vel += control.linearAcc * params.deltaT;
      f.angularVel += control.angularAcc * params.deltaT;
    }
    default: {
    }
  }

  // kinematic update
  f.pos += f.vel * params.deltaT;
  f.angle = normalizeAngle(f.angle + f.angularVel * params.deltaT);

  // Bounce off the boundary.
  let under = (f.pos < params.minBound) & (f.vel < vec2());
  let over = (f.pos > params.maxBound) & (f.vel > vec2());
  f.vel = select(f.vel, -f.vel * params.boundaryBounceFactor, under | over);
  f.pos = clamp(f.pos, params.minBound, params.maxBound);

  // Write back
  particlesB.particles[index].pos = f.pos;
  particlesB.particles[index].vel = f.vel;
  particlesB.particles[index].angle = f.angle;
  particlesB.particles[index].angularVel = f.angularVel;
}

fn particleReferenceFrame(particle : Particle) -> ReferenceFrame {
  return ReferenceFrame(particle.pos, particle.vel, particle.angle, particle.angularVel);
}

fn particleType(particle : Particle) -> u32 {
  return (particle.metadata >> 8) & 0xff;
}

fn particleTeam(particle : Particle) -> u32 {
  return particle.metadata & 0xff;
}

fn flock(particle : Particle, selfIdx : u32) -> vec2f {
  var vPos = particle.pos;
  var vVel = particle.vel;
  var cMass = vec2(0.0);
  var cVel = vec2(0.0);
  var colVel = vec2(0.0);
  var cMassCount = 0u;
  var cVelCount = 0u;

  let myTeam = particleTeam(particle);

  for (var i = 0u; i < arrayLength(&particlesA.particles); i++) {
    let other = particlesA.particles[i];
    if (i == selfIdx || particleType(other) != bodyTypeShip) {
      continue;
    }
    let pos = other.pos.xy;
    let vel = other.vel.xy;
    let dPos = pos - vPos;
    let dist = length(dPos);
    if (dist < params.avoidDistance) {
      colVel -= dPos;
    }
    if (particleTeam(other) == myTeam) {
      if (dist < params.cMassDistance) {
        cMass += pos;
        cMassCount++;
      }
      if (dist < params.cVelDistance) {
        cVel += vel;
        cVelCount++;
      }
    }
  }
  if (cMassCount > 0) {
    cMass = (cMass / vec2(f32(cMassCount))) - vPos;
  }
  if (cVelCount > 0) {
    cVel /= f32(cVelCount);
  }
  vVel += (colVel * params.avoidScale) + (cMass * params.cMassScale) + (cVel * params.cVelScale);

  // clamp velocity for a more pleasing simulation
  // TODO: make upper bound a param.
  return normalize(vVel) * clamp(length(vVel), 0.0, 100.0);
}

fn updateMissile(current : ReferenceFrame, selfIdx : u32, targetIdx : u32) -> Control {
  if (targetIdx >= arrayLength(&particlesA.particles)) {
    return Control();
  }

  var vPos = current.pos;
  var vVel = current.vel;
  let targetP = particlesA.particles[targetIdx];
  let targetF = particleReferenceFrame(targetP);

  let targetPos = targetF.pos;
  let targetVel = targetF.vel;
  let targetVec = current.pos - targetPos;
  let targetDist = length(targetVec);
  let targetDir = normalize(targetVec);  // TODO: handle zero targetDist

  let desiredDir = targetDir;
  let desiredDist = clamp(targetDist, 0.0, 0.0);      // TODO: this would be min/max distance
  let desiredPos = targetPos + desiredDir * desiredDist;
  let desiredAngle = angleOf(current.vel, current.angle); // angleOf(-targetDir);   // TODO: for ships use -targetDir  
  let desired = ReferenceFrame(desiredPos, targetVel, desiredAngle, 0.0);

  let rel = referenceFrameSub(desired, current);
  // Transform into the missile's coordinate system.
  let localRel = referenceFrameRotate(rel, -current.angle);
  let forward = vec2f(0.0, 1.0);
  var localLinAcc = proNav2D(localRel.pos, localRel.vel, forward);

	// Accelerate forward as fast as possible while staying under MaxSpeed (with respect to target).
  let speed = -localRel.vel.y;
  if (params.maxSpeed == 0 || speed < params.maxSpeed) {
    let maxAcc = min((params.maxSpeed - speed) / params.deltaT, params.maxAcc);
    localLinAcc.y += maxAcc;
  }

  // Limit acceleration
  let l = length(localLinAcc);
  if (l > params.maxAcc) {
    localLinAcc *= params.maxAcc / l;
  }

  let linAcc = rotVec(localLinAcc, current.angle);
  let angAcc = computeTurnAcceleration(rel.angle, rel.angularVel);

  return Control(linAcc, angAcc);
}

fn proNav2D(relPos : vec2f, relVel : vec2f, forward : vec2f) -> vec2f {
	// TODO: most of the math simplifies with z=0.
	return proNav3D(vec3f(relPos, 0.0), vec3f(relVel, 0.0), vec3f(forward, 0.0)).xy;
}

fn proNav3D(relPos : vec3f, relVel : vec3f, forward : vec3f) -> vec3f {
  let omega = cross(relPos, relVel) / dot(relPos, relPos);
	return cross(forward * -proNavGain * length(relVel), omega);
}

fn computeTurnAcceleration(relAng : f32, relAngVel : f32) -> f32 {
  let maxAcc = params.maxAngAcc;
	// Compute the maximum velocity we can turn at and still stop in time.
	// Given v^2 = u^2 + 2as, assuming v=0 then u = sqrt(-2as).
	// The most we can accelerate in this frame is (sqrt(2as)-u)/t.
	// https://physics.stackexchange.com/questions/312692.
  let absAngDiff = abs(relAng);
  let absAngSign = sign(relAng);
	let maxBrakingVel = sqrt(2.0 * maxAcc * absAngDiff) * absAngSign;
	return clamp((maxBrakingVel + relAngVel) / params.deltaT, -maxAcc, maxAcc);
}