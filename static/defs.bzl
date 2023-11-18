"""Bazel rules for copying files from the Go toolchain."""
load("@rules_go//go:def.bzl", "go_context")

def _go_copy_sdk_file_impl(ctx):
    go = go_context(ctx)
    cmd = "cp $({bin} env GOROOT)/{relpath} {out}".format(
        bin = go.go.path,
        relpath = ctx.attr.sdkfile,
        out = ctx.outputs.out.path,
    )
    ctx.actions.run_shell(
        outputs = [ctx.outputs.out],
        inputs = [],
        tools = [go.go],
        command = cmd,
        mnemonic = "GoCopySDKFile",
    )
    return [DefaultInfo(files = depset([ctx.outputs.out]))]

go_copy_sdk_file = rule(
    implementation = _go_copy_sdk_file_impl,
    doc = "go_copy_sdk_file fetches a file from the golang SDK",
    attrs = {
        "sdkfile": attr.string(mandatory = True),
        "out": attr.output(mandatory = True),
        "_go_context_data": attr.label(
            default = "@rules_go//:go_context_data",
        ),
    },
    toolchains = ["@rules_go//go:toolchain"],
)

def _gzip_file_impl(ctx):
    cmd = "gzip -c {src} > {out}".format(
        src = ctx.file.src.path,
        out = ctx.outputs.out.path,
    )
    ctx.actions.run_shell(
        outputs = [ctx.outputs.out],
        inputs = [ctx.file.src],
        command = cmd,
        mnemonic = "GZipFile",
    )
    return [DefaultInfo(files = depset([ctx.outputs.out]))]

gzip_file = rule(
    implementation = _gzip_file_impl,
    doc = "gzip_file gzips a file",
    attrs = {
        "src": attr.label(mandatory = True, allow_single_file = True),
        "out": attr.output(mandatory = True),
    },
)
