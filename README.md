# gowebgpu
An experiment with Go + WASM + WebGPU

## Prerequisites

- **Go 1.21+**
- **Bazel 9.x** (via [Bazelisk](https://github.com/bazelbuild/bazelisk))
- **gcc** (required by Bazel's CC toolchain)

## Local Testing

Natively:

```bash
bazel run :gowebgpu -- --port=9090 --tls
```

With docker:

```bash
bazel build //:gowebgpu_tarball
docker load --input $(bazel cquery --output=files //:gowebgpu_tarball)
docker run --rm -p 9090:80 gowebgpu:latest
```
