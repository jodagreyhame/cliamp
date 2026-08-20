# go-librespot overlays

These files replace the CGO Vorbis/FLAC decoders and the Unix-only pipe
open in [go-librespot](https://github.com/devgianlu/go-librespot) v0.7.1
so Spotify playback compiles without libogg, libvorbis, libflac, or
Unix `O_NONBLOCK` on the output pipe.

From the repo root:

```
go run scripts/setup_librespot.go
```

That clones v0.7.1 into `third_party/librespot` (gitignored) and copies
this directory over it. `go.mod` replaces the module with that checkout.
Run it once before `go build` / `go test`. `make build` and `make test`
run it first.
