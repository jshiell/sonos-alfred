# Sonos for Alfred

Go workflow (`son` keyword) controlling a single Sonos hub over local UPnP/SOAP. Design and plan: `plan-v1.md`.

## Verify gate

```
go vet ./... && go test ./...
```

Go is pinned in `mise.toml` (run via the mise shim; arm64 only).

## Known issues

- Inside the nono sandbox, mise prints `WARN tracking config: failed to ln -sf ... Operation not permitted` and `mise trust` fails. The sandbox cannot write `~/.local/state/mise`. The warning is harmless: `go` still runs the pinned version. Don't try to fix it from inside the sandbox.
- Real speakers, from inside the nono sandbox: an earlier session saw every Go HTTP request time out while `/usr/bin/curl` worked; a later one reached them fine from Go. Treat it as intermittent; the cause is not verified, and it may be the fresh-binary stall below. Tests use the fake speaker and are unaffected. If Go can't connect, try `/usr/bin/curl`, or ask the user to run the command outside the sandbox.
- A freshly built binary can stall on its first network calls (5s timeouts, so `refresh` shows "Can't reach Sonos … context deadline exceeded") for a minute or two, then run normally (0.1s). Seen after `go build` and packaging; the cause is not verified. Retry before treating it as a bug.
- Fixtures in `testdata/` are sanitised captures of a real household: addresses are `192.0.2.x`, speaker IDs are `RINCON_1111…`, and titles are made up. Sanitise any new capture the same way before committing it.
- `spikes/` is throwaway code and may be excluded from the gate once fixtures are captured.
