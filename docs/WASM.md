# WebAssembly Targets

Tya can build simple WebAssembly modules through Zig while preserving the
compile-to-C backend.

Native builds remain the default:

```sh
tya build src/main.tya -o app
tya build --target native src/main.tya -o app
```

WASI builds produce a `.wasm` module:

```sh
tya doctor wasm
tya build --target wasm32-wasi examples/wasm/hello.tya -o /tmp/hello.wasm
```

Browser builds produce a `.wasm` module and a JavaScript loader next to it:

```sh
tya build --target wasm32-browser examples/wasm/hello.tya -o /tmp/hello-browser/hello.wasm
```

The first WebAssembly target layer supports stdout-oriented smoke programs and
rejects native packages for WASM builds. Browser builds also reject filesystem
and process-oriented imports. `tya run` remains native-only.
