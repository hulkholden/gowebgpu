struct Particles {
  particles : array<Particle>,
}
@binding(0) @group(0) var<uniform> params : SimParams;
@binding(1) @group(0) var<storage, read> particlesA : Particles;
@binding(2) @group(0) var<storage, read_write> particlesB : Particles;

@binding(3) @group(0) var<storage, read_write> contacts : ContactsContainer;

const pi = 3.14159265359;
const twoPi = 2 * pi;

const proNavGain = 3.0;

const bodyTypeNone = 0u;
const bodyTypeShip = 1u;
const bodyTypeMissile = 2u;

fn addContact(aIdx : u32, bIdx : u32) {
  let contactIdx = atomicAdd(&contacts.count, 1);
  if (contactIdx < contacts.capacity) {
    contacts.elements[contactIdx] = Contact(aIdx, bIdx);
  }
}

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
  var particle = particlesA.particles[index];

  let f = particleReferenceFrame(particle);

  var control = Control();
  switch particleType(particle) {
    case bodyTypeNone: {
    }
    case bodyTypeShip: {
      control = flock(f, particleTeam(particle), index);
    }
    case bodyTypeMissile: {
      if (particle.targetIdx < 0) {
        particle.targetIdx = findTarget(particle);
        if (particle.targetIdx < 0) {
          break;
        }
      }
      // Reset on contact.
      let targetP = particlesA.particles[particle.targetIdx];
      if (particle.age > params.maxMissileAge || distance(particle.pos, targetP.pos) < 10.0) {
        particle.pos = 2.0 * (rand22(particle.pos) - 0.5) * 1000.0;
        particle.vel = 2.0 * (rand22(particle.vel) - 0.5) * 0.0;
        particle.angle = 0.0;
        particle.angularVel = 0.0;
        particle.age = 0.0;
      }
      control = updateMissile(f, index, particle.targetIdx);
    }
    default: {
    }
  }

  // kinematic update
  particle.age += params.deltaT;

  particle.vel += control.linearAcc * params.deltaT;
  particle.pos += particle.vel * params.deltaT;

  particle.angularVel += control.angularAcc * params.deltaT;
  particle.angle = normalizeAngle(particle.angle + particle.angularVel * params.deltaT);

  // Bounce off the boundary.
  let under = (particle.pos < params.minBound) & (particle.vel < vec2());
  let over = (particle.pos > params.maxBound) & (particle.vel > vec2());
  particle.vel = select(particle.vel, -particle.vel * params.boundaryBounceFactor, under | over);
  particle.pos = clamp(particle.pos, params.minBound, params.maxBound);

  if (particleType(particle) == bodyTypeShip) {
    // clamp velocity for a more pleasing simulation
    // TODO: make upper bound a param.
    particle.vel =  normalize(particle.vel) * clamp(length(particle.vel), 0.0, 100.0);
  }

  // Write back
  particlesB.particles[index] = particle;
}

fn findTarget(particle : Particle) -> i32 {
	let selfTeam = particleTeam(particle);
  let selfType = particleType(particle);
	if (selfType != bodyTypeMissile) {
		return -1;
	}

	let wantType = select(bodyTypeShip, bodyTypeMissile, selfTeam == 2);
	var closestIdx = -1;
	var closestDist = 0.0;
  for (var idx = 0u; idx < arrayLength(&particlesA.particles); idx++) {
    let other = particlesA.particles[idx];
		if (particleTeam(other) == selfTeam || particleType(other) != wantType) {
			continue;
		}
    let dist = distance(particle.pos, other.pos);
		if (closestIdx < 0 || dist < closestDist) {
			closestDist = dist;
			closestIdx = i32(idx);
		}
	}
	return closestIdx;
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

fn flock(current : ReferenceFrame, selfTeam : u32, selfIdx : u32) -> Control {
  var cMass = vec2(0.0);
  var cVel = vec2(0.0);
  var colVel = vec2(0.0);
  var cMassCount = 0u;
  var cVelCount = 0u;

  for (var i = 0u; i < arrayLength(&particlesA.particles); i++) {
    let other = particlesA.particles[i];
    if (i == selfIdx || particleType(other) != bodyTypeShip) {
      continue;
    }
    let pos = other.pos.xy;
    let vel = other.vel.xy;
    let dPos = pos - current.pos;
    let dist = length(dPos);
    if (dist < params.avoidDistance) {
      colVel -= dPos;
    }
    if (particleTeam(other) == selfTeam) {
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
    cMass = (cMass / vec2(f32(cMassCount))) - current.pos;
  }
  if (cVelCount > 0) {
    cVel /= f32(cVelCount);
  }
  
  let dVel = (colVel * params.avoidScale) + (cMass * params.cMassScale) + (cVel * params.cVelScale);
  let linAcc = dVel / params.deltaT;

  // Set the desired reference frame to the current state, attempting to orient with the velocity vector.
  // TODO: we could ignore linear component of this - maybe the compiler does that for us?
  let desired = ReferenceFrame(current.pos, current.vel, angleOf(current.vel, current.angle), 0.0);
  let rel = referenceFrameSub(desired, current);
  let angAcc = computeTurnAcceleration(rel.angle, rel.angularVel);

  return Control(linAcc, angAcc);
}

fn updateMissile(current : ReferenceFrame, selfIdx : u32, targetIdx : i32) -> Control {
  let targetP = particlesA.particles[targetIdx];
  let targetF = particleReferenceFrame(targetP);

  let targetVec = current.pos - targetF.pos;
  let targetDist = length(targetVec);
  let targetDir = normalize(targetVec);  // TODO: handle zero targetDist

  let desiredDir = targetDir;
  let desiredDist = clamp(targetDist, 0.0, 0.0);      // TODO: this would be min/max distance
  let desiredPos = targetF.pos + desiredDir * desiredDist;
  let desiredAngle = angleOf(current.vel, current.angle); // angleOf(-targetDir);   // TODO: for ships use -targetDir  
  let desired = ReferenceFrame(desiredPos, targetF.vel, desiredAngle, 0.0);

  let rel = referenceFrameSub(desired, current);
  // Transform into the missile's coordinate system.
  let localRel = referenceFrameRotate(rel, -current.angle);

  var localLinAcc = vec2f(0, 0);
  // Apply proportional navigation to track towards the target.
  localLinAcc.x = proNav2D(localRel.pos, localRel.vel);
	// Accelerate forward as fast as possible while staying under MaxSpeed (with respect to target).
  if (params.maxSpeed == 0) {
    localLinAcc.y = params.maxAcc;
  } else {
    // Relative velocity is negative as we're closing on the target.
    let speed = -localRel.vel.y;
    if (speed < params.maxSpeed) {
      let maxAcc = min((params.maxSpeed - speed) / params.deltaT, params.maxAcc);
      localLinAcc.y = maxAcc;
    }
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

// A version of https://en.wikipedia.org/wiki/Proportional_navigation simplified for 2D.
fn proNav2D(r : vec2f, v : vec2f) -> f32 {
  return -proNavGain * perpDot(r, v) * length(v) / dot(r, r);
}

fn perpDot(a: vec2f, b: vec2f) -> f32 {
	return a.x*b.y - a.y*b.x;
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

// https://gist.github.com/munrocket/236ed5ba7e409b8bdf1ff6eca5dcdc39
fn hash22(p: vec2u) -> vec2u {
    var v = p * 1664525u + 1013904223u;
    v.x += v.y * 1664525u; v.y += v.x * 1664525u;
    v ^= v >> vec2u(16u);
    v.x += v.y * 1664525u; v.y += v.x * 1664525u;
    v ^= v >> vec2u(16u);
    return v;
}

fn rand22(f: vec2f) -> vec2f { return vec2f(hash22(bitcast<vec2u>(f))) / f32(0xffffffff); }