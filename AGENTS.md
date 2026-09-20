# Sonos for Alfred

Go workflow (`son` keyword) controlling a single Sonos hub over local UPnP/SOAP. Design and plan: `plan-v1.md`.

## Verify gate

```
go vet ./... && go test ./...
```

Go is pinned in `mise.toml` (run via the mise shim; arm64 only).

## Known issues

- Inside the nono sandbox, mise prints `WARN tracking config: failed to ln -sf ... Operation not permitted` and `mise trust` fails. The sandbox cannot write `~/.local/state/mise`. The warning is harmless: `go` still runs the pinned version. Don't try to fix it from inside the sandbox.
- Inside the nono sandbox, Go programs cannot talk to real speakers: the TCP connect to port 1400 succeeds, but every HTTP request gets no bytes and times out. `/usr/bin/curl` works, and the same binary works outside the sandbox (refresh in 0.26s). The cause is not verified. Tests use the fake speaker and are unaffected. To check against real speakers, ask the user to run the command outside the sandbox.
- `spikes/` is throwaway code and may be excluded from the gate once fixtures are captured.
