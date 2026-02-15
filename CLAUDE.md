# CLAUDE.md

## Project Overview

gowebgpu is an experimental Go + WebAssembly (WASM) + WebGPU application that renders GPU-accelerated particle simulations in the browser. Go code compiles to WASM, which runs in the browser and interfaces with the WebGPU API for GPU compute and rendering.

The project contains two example simulations:
- **battle** — Multi-team space battle with ships and missiles (4000 particles, 7 compute passes)
- **boids** — Classic boid flocking algorithm (20000 particles, 1 compute pass)

## Repository Structure

```
gowebgpu/
├── main.go                    # HTTP server entry point
├── client/                    # WASM client code (compiles to client.wasm)
│   ├── main.go                # WASM entry point, loads WebGPU device
│   ├── browser/               # Browser API wrappers (requestAnimationFrame)
│   ├── engine/                # GPU abstraction layer (buffers, compute passes, shaders)
│   └── examples/              # Demo applications
│       ├── battle/            # Space battle simulation (.go + .wgsl shaders)
│       └── boids/             # Boid flocking simulation (.go + .wgsl shaders)
├── common/                    # Shared utilities (no WASM dependencies)
│   ├── math32/                # Float32 math wrappers
│   ├── vmath/                 # Vector types (V2, V3, V4)
│   └── wgsltypes/             # Go struct ↔ WGSL type bridge (reflection-based)
├── static/                    # Embedded static assets (JS, CSS, WASM binary)
│   └── defs.bzl               # Custom Bazel rules (go_copy_sdk_file, gzip_file)
└── templates/                 # HTML templates
```

## Build System

**Primary build system: Bazel** (with bzlmod enabled via `.bazelrc`).

### Prerequisites

- **Go 1.21+**
- **Bazel 7.x** (via [Bazelisk](https://github.com/bazelbuild/bazelisk)) — pinned in `.bazelversion`
- **gcc** (required by Bazel's CC toolchain)

### Key Commands

```bash
# Run the server natively (opens on port 9090)
bazel run :gowebgpu -- --port=9090

# Build all targets explicitly
# Note: `bazel build //...` fails because it includes WASM-only library targets
# (e.g. //client:client_lib) which can't compile for the host platform.
bazel build //:gowebgpu //client //:gowebgpu_linux //static //common/...

# Run all tests
bazel test //...

# Run specific tests
bazel test //common/wgsltypes:wgsltypes_test

# Build Docker image
bazel build //:gowebgpu_tarball
docker load --input $(bazel cquery --output=files //:gowebgpu_tarball)
docker run --rm -p 9090:80 gowebgpu:latest

# Regenerate BUILD files after adding/removing Go files
bazel run :gazelle

# Sync Bazel deps with go.mod
bazel run :gazelle-update-repos
```

### Build Targets

| Target | Description |
|--------|-------------|
| `:gowebgpu` | Native server binary |
| `:gowebgpu_linux` | Linux AMD64 binary (for container) |
| `:gowebgpu_tarball` | OCI container image tarball |
| `//client` | WASM client binary (client.wasm) |
| `//static` | Embedded static assets library |

### WASM Compilation

The client package compiles to WASM via Bazel's `go_binary` with `goarch = "wasm"` and `goos = "js"`. The resulting `client.wasm` is copied and gzipped into the `static/` package for embedding. The `wasm_exec.js` runtime shim is extracted from the Go SDK automatically.

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/mokiat/wasmgpu` | WebGPU Go bindings (uses fork at `github.com/hulkholden/wasmgpu`) |
| `github.com/mokiat/gog` | Go utilities (`opt` package for option types) |
| `github.com/mroth/weightedrand/v2` | Weighted random selection (particle spawning) |
| `github.com/google/go-cmp` | Test comparison utilities |

The `go.mod` contains a `replace` directive pointing `mokiat/wasmgpu` to a custom fork.

## Testing

- Tests use Go's standard `testing` package with `go-cmp` for diff assertions.
- Test files live alongside source files (e.g., `struct_test.go` next to `struct.go`).
- Run tests via `bazel test //...` (not `go test`).
- Currently, tests exist only in `common/wgsltypes/`.

## Code Conventions

### Go Style
- Go 1.21 with generics used heavily for type-safe GPU buffers (`GPUBuffer[T]`, `DebugBuffer[T]`).
- Unexported struct fields (lowercase) are standard — the structs map directly to GPU buffer layouts where field ordering and alignment matter.
- Functional options pattern for buffer configuration (`WithVertexUsage()`, `WithCopySrcUsage()`).
- `Must*` prefix for functions that panic on error (e.g., `MustRegisterStruct`).
- `//go:embed` for shader code and static assets.
- No explicit error wrapping style; errors use `fmt.Errorf` with `%v`.

### WGSL Shaders
- Shader files (`.wgsl`) live alongside their Go example code.
- WGSL struct definitions are generated from Go structs at runtime via `wgsltypes` and prepended to shader code as a "prologue".
- Compute shaders use workgroup size of 64 (hardcoded in both Go and WGSL).

### Struct Tags
- `atomic:"true"` — marks a field as `atomic<T>` in WGSL.
- `runtimeArray:"true"` — marks an array field as a runtime-sized array in WGSL.

### Architecture Patterns
- **Go ↔ WGSL type bridge**: `wgsltypes` uses reflection to map Go structs to WGSL struct definitions, validating alignment requirements.
- **Multi-pass compute pipeline**: `ComputePassFactory` creates reusable compute passes from a single shader module with different entry points.
- **`ComputePassBuffer` interface**: GPU buffers implement this interface to provide struct definitions and bind group entries to the compute pipeline factory.
- **Animation loop**: `engine.InitRenderCallback` sets up a `requestAnimationFrame` loop via JS interop.

## CI/CD

Google Cloud Build pipeline (`cloudbuild.yaml`):
1. Bazel builds the OCI tarball
2. Docker loads and tags the image
3. Pushes to Google Artifact Registry

## Key Implementation Details

- The server embeds all static assets and templates into the binary using `embed.FS`.
- The WASM binary is served gzip-compressed; the server has middleware to handle `Accept-Encoding: gzip`.
- Client WASM waits for JavaScript to initialize WebGPU context before proceeding (`waitForExports` polling loop).
- GPU buffer data is shared between Go and WGSL by ensuring identical memory layouts — `wgsltypes` validates field alignment against WGSL requirements at struct registration time.
- The `client/main.go` currently hardcodes running the `battle` example.
