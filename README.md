# gowebgpu
An experiment with Go + WASM + WebGPU

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
