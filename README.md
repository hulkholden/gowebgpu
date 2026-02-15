# gowebgpu
An experiment with Go + WASM + WebGPU

## Prerequisites

- **Go 1.21+**
- **Bazel 7.x** (via [Bazelisk](https://github.com/bazelbuild/bazelisk))
- **gcc** (required by Bazel's CC toolchain)

## Local Testing

Natively:

```bash
bazel run :gowebgpu -- --port=9090
```

With docker:

```bash
bazel build //:gowebgpu_tarball
docker load --input $(bazel cquery --output=files //:gowebgpu_tarball)
docker run --rm -p 9090:80 gowebgpu:latest
```

## Build Notes

`bazel build //...` does not work because the wildcard includes WASM-only library
targets (e.g. `//client:client_lib`) which fail to compile for the host platform.
To build all targets explicitly:

```bash
bazel build //:gowebgpu //client //:gowebgpu_linux //static //common/...
```
