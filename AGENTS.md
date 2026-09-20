# Sonos for Alfred

Go workflow (`son` keyword) controlling a single Sonos hub over local UPnP/SOAP. Design and plan: `plan-v1.md`.

## Verify gate

```
go vet ./... && go test ./...
```

Go is pinned in `mise.toml` (run via the mise shim; arm64 only).

## Known issues

- Inside the nono sandbox, mise prints `WARN tracking config: failed to ln -sf ... Operation not permitted` and `mise trust` fails. The sandbox cannot write `~/.local/state/mise`. The warning is harmless: `go` still runs the pinned version. Don't try to fix it from inside the sandbox.
- `spikes/` is throwaway code and may be excluded from the gate once fixtures are captured.
