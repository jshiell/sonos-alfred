# Sonos for Alfred — Design + Implementation Plan (v1, post-review)

## Context
Existing Alfred Sonos workflows are thin: one keyword per action, no favorites/grouping/rooms UX, discovery failures (VPN), or they drive the Sonos Mac app via AppleScript (abandoned 2020). Raycast's Sonos extension and `sonoscli` (Go, MIT) show the better model: coordinator-aware control, an "active group", favorites, scenes. Goal: a fast, single-hub Alfred workflow for personal use.

This revision folds in a plan review (see "Changes from review"). API details are cross-checked against primary sources where noted; anything else is marked unverified.

## Decisions (yours)
| Topic | Decision |
|---|---|
| Sources | Apple Music, Sonos Radio/TuneIn (both reachable via Favorites) |
| v1 music scope | Favorites + Sonos Playlists + Queue. **No search in v1.** |
| Audience | Just me: no signing/notarization/update mechanism, plain config |
| Language | Go (via mise) |
| Alfred output layer | **Own Script Filter JSON structs.** AwGo dropped as output layer (changed after review) |
| Architecture | **arm64 only** (changed after review; no `lipo`) |
| Setup | 1–3 rooms, rarely grouped; has a stereo pair / soundbar+sub. No VPN concern. |
| Layout | One hub keyword (`son`) with drill-down |
| Filtering | **Per-keystroke Script Filter; Go does the matching; "Alfred filters results" off; Alfred `cache` key unused** |
| Targeting | Persistent active group; default = whichever group is playing; room picker overrides |
| v1 extras | Shuffle / repeat / sleep timer only. **No grouping, scenes, line-in in v1.** |
| Enter on Favorite/Playlist | Replace queue + play. ⌘ = add to end. ⌥ = play next |
| Volume | Type a number (`son vol 35`) + ± step items (configurable step, default 5). Group volume default |
| Feedback | Silent on success; notification on error (mechanism decided by spike S3) |
| Dev loop | mise task builds `.alfredworkflow`; you import it |

## Stack
- Go, standard library plus a small set of dependencies (XML/HTTP are stdlib). Installed via mise.
- Own Script Filter JSON structs (format is small and documented: https://www.alfredapp.com/help/workflows/inputs/script-filter/json/).
- Background refresh: ~60 lines of our own (detached process via `Setpgid`, lock file), modelled on AwGo's `background.go` (MIT).
- Thin hand-written SOAP client over the local UPnP API. References: svrooij docs, SoCo source, `sonoscli` (MIT).
- arm64 binary bundled in the workflow: no runtime or PATH dependency.
- Tests: `go test` with a **strict** fake speaker (see 2.0), fixtures captured from your real speakers. TDD per your CLAUDE.md.

## Architecture
One binary, three roles:
- `sonos-alfred filter <query>` → emits Script Filter JSON. **Reads cache only; makes zero network calls.**
- `sonos-alfred do <encoded-action>` → performs an action against the coordinator; on success also invalidates affected cache entries.
- `sonos-alfred refresh <scope>` → talks to the speakers and rewrites the cache atomically. Spawned detached by `filter` when data is stale. Single-flight via lock file.

Packages:
- `sonos/` — discovery, topology, AVTransport, GroupRenderingControl, ContentDirectory. **Behind an interface**, so a REST backend (`https://{ip}:1443/api`, undocumented in the svrooij docs) could replace UPnP if Sonos removes it.
- `state/` — cache dir (atomic writes): topology, active group (stored as player UUID), last-known now-playing, Favorites/Playlists/Queue lists, last refresh outcome (success / unreachable).
- `hub/` — pure functions: state + query in → neutral items out. Most unit tests live here.
- `alfredjson/` — the single renderer from neutral items to Script Filter JSON.
- `main` — subcommand wiring.

Alfred graph: Script Filter (`son`, "Alfred filters results" off) → Run Script (`do`) → error notification (mechanism per S3).

### Latency contract (per-keystroke)
- `filter` performs **no blocking network I/O, ever.** Topology, active group, now-playing, and the Favorites/Playlists/Queue lists all come from cache.
- If a cache entry is stale, `filter` renders what it has and spawns a detached `refresh` (single-flight; a lock file with stale-lock handling). It sets `rerun` **only while a refresh is pending**, and caps the number of reruns, so it can't poll the speaker indefinitely while Alfred is open.
- Cold cache (first ever run): one "Loading Sonos…" item plus a refresh, then render.
- SSDP discovery (can take seconds) only ever runs inside `refresh`.
- `do` invalidates the entries it changes (e.g. a favorite play invalidates the queue and now-playing), so the next `filter` shows fresh state.

### Target resolution
1. Stored active-group player UUID → find the **group that currently contains that player** → target that group's **current coordinator**. (A player that was regrouped in the Sonos app is still present, but no longer a coordinator; commands to a non-coordinator fail.)
2. If nothing stored, or the player is gone: whichever group the cached transport state says is playing.
3. Otherwise the first group.

### `hub` → `do` argument encoding
One documented, versioned string format (`verb:payload`, payload URL-escaped), defined in increment 4.0 with a round-trip test, before any menu item is built. `do` parses the same format.

## Hub layout
Top level: **now-playing row**, volume row, then Favorites, Playlists, Queue, Rooms, Shuffle/Repeat, Sleep timer. Typing filters fuzzily across everything. `vol 35` yields a "Set volume 35" item.

| Item | Enter | ⌘ | ⌥ |
|---|---|---|---|
| Favorite / Playlist | replace queue + play | add to end | play next |
| Queue track | jump to track | | |
| Room | set active group | | |
| Now playing | play/pause | next | previous |

⌘/⌥ on a favorite that can't be enqueued (radio streams, if S2 confirms) are shown invalid or fall back to replace. The choice is made from what S2 actually records, not assumed.

## Sonos API facts (cross-checked in review against svrooij docs / SoCo source unless flagged)
- `AddURIToQueue(EnqueuedURI, EnqueuedURIMetaData, DesiredFirstTrackNumberEnqueued, EnqueueAsNext)` — `EnqueueAsNext` is an **argument**, not an action.
- `RemoveAllTracksFromQueue`, `Seek(Unit=TRACK_NR, Target=<1-based>)`.
- **Queue jump** = `SetAVTransportURI(x-rincon-queue:<coordinatorUID>#0)` **then** `Seek TRACK_NR` (SoCo `play_from_queue`). Bare `Seek` fails if the group is on a stream or line-in.
- Play mode is a single enum: `NORMAL`, `REPEAT_ALL`, `REPEAT_ONE`, `SHUFFLE_NOREPEAT`, `SHUFFLE`, `SHUFFLE_REPEAT_ONE`. **`SHUFFLE` = shuffle + repeat-all.** Read via `GetTransportSettings` → `PlayMode`.
- Sleep timer: `ConfigureSleepTimer(NewSleepTimerDuration)` as `hh:mm:ss`, empty string cancels; read back with `GetRemainingSleepTimerDuration`.
- Group volume: `GetGroupVolume`, `SetGroupVolume(DesiredVolume)`, `SetRelativeGroupVolume(Adjustment)` (single call, clamps device-side, returns `NewVolume`), `SnapshotGroupVolume`. **Whether a snapshot must be taken explicitly is unverified → S2.**
- ContentDirectory `Browse`: Favorites `FV:2`, Playlists `SQ:`, Queue **`Q:0`** (SoCo's value; svrooij lists `Q:`). Favorites carry `r:resMD` (the referenced item's metadata). Play the `<res>` URI with `r:resMD` as `CurrentURIMetaData`.
- Topology (`GetZoneGroupState`) has **two shapes**: a stereo-pair partner is a top-level member with `Invisible="1"`; a home-theatre sub/surround is a nested `<Satellite>` inside a `ZoneGroupMember`. Your firmware's actual shape is captured in S1.
- Escaping: DIDL metadata is XML-escaped inside the SOAP body; over-decoding favorite URIs is a known cause of UPnP 800 faults. Tests must assert exact outgoing bytes.

## Proposed defaults (my choice, not asked. Object if wrong)
- Topology cache TTL ~5 min; now-playing ~2 s; lists ~30 s; all refreshed in the background.
- Optional "speaker IP" in workflow config as a discovery fallback. `/24` subnet scan deferred (no VPN).
- Bundle ID `org.infernus.sonos-alfred`; assumes **Alfred 5 + Powerpack** (exact minor version doesn't matter, since `cache` is unused).
- Sleep timer: presets (15/30/60 min, off).

## Unverified → resolved by spikes before feature work
1. Local UPnP still works on your speakers/firmware (one community post claims gradual S2 deprecation; unconfirmed).
2. Whether radio favorites can be enqueued. SoCo imposes no such restriction; reports of fault 800 exist. Record the exact fault.
3. Whether `SnapshotGroupVolume` must be called explicitly.
4. Stereo-pair and soundbar+sub topology shape on your firmware.
5. How Alfred surfaces a Run Script failure. The Alfred docs are silent on exit status/stderr.
6. Whether Go's arm64 output is already ad-hoc signed and runs cleanly from an imported workflow.

## Out of scope for v1
SMAPI search (needs its own spike incl. Apple Music auth), grouping/scenes, line-in/TV, TTS/announcements, Universal Actions, cloud Control API, EQ/alarms, amd64, updates/notarization.

## Implementation plan

Rules (from your CLAUDE.md): one test → one implementation → one commit per increment; commit only when green; never modify a passing test to make new code compile; never push. If a commit fails to sign, stop and ask you to approve the 1Password prompt. Run the `verify` skill before declaring any phase done. Each Red step first checks the action/args against the svrooij docs or a captured fixture.

### Phase 0 — Scaffold (not TDD)
0.1 `git init`; `.gitignore`; `mise.toml` pinning Go; `go mod init sonos-alfred`.
0.2 `AGENTS.md` declaring the verify gate: `go vet ./... && go test ./...`.
0.3 **Ask you now (parallel, longest-lead item):** export the skeleton workflow (Script Filter `son` → Run Script → Post Notification) into the project dir.

### Phase 1 — Spikes (throwaway, flagged; live in `spikes/`, deleted after fixtures are captured)
The sandbox may block LAN access. If so, I stop and give you the single command to run with `!`.
- **S1 Discovery + topology.** SSDP M-SEARCH + `GetZoneGroupState`; save raw XML to `testdata/`. Must capture **both** shapes: your stereo pair (`Invisible="1"` members?) and your soundbar+sub (nested `<Satellite>`?). Also time SSDP.
- **S2 Favorites + volume.** `Browse FV:2` (save XML). Play one Apple Music and one Radio favorite via `<res>` + `r:resMD`. Try `AddURIToQueue` on the radio favorite and **record the exact SOAP fault code**. Change one member's volume in the Sonos app, then `SetGroupVolume`, and observe whether the ratio is respected (answers the snapshot question). Save the outgoing request bodies as fixtures.
- **S3 Packaging + Alfred behaviour.** arm64 build → `codesign -dv` → drop into the skeleton workflow → confirm Alfred runs it. Confirm a hand-written Script Filter JSON with `mods` renders correctly. Determine the error-notification mechanism: (a) does a non-zero exit surface anything, (b) does Run Script output feeding Post Notification work. **Fallback** if neither is clean: `do` posts its own notification via `osascript`.
- **Gate:** update this document with the findings. Stop if UPnP is dead (the backend choice would change).

### Phase 2 — `sonos/` (strict fake speaker, fixtures from S1/S2)
2.0 **Strict fake speaker** (test infrastructure, tested on its own): rejects unexpected actions, wrong argument order/values, and out-of-sequence calls; assertions on exact request bytes.
2.1 SOAP call: correct `SOAPACTION` + body; parses response args; SOAP faults become typed errors.
2.1b Escaping: DIDL metadata round-trips through the envelope; exact bytes asserted against an S2 fixture.
2.2a Parse `ZoneGroupState` → groups + coordinators.
2.2b Hide `Invisible="1"` members (stereo pair).
2.2c Hide nested `<Satellite>` elements (sub/surrounds).
2.3 SSDP response parsing → speaker IP (network I/O behind an interface; hand-verified).
2.4 Transport: play, pause, next, previous, transport state.
2.5 Now-playing (`GetPositionInfo`): tolerate empty metadata.
2.6a Group volume get/set. 2.6b ± step via `SetRelativeGroupVolume`. 2.6c snapshot behaviour per S2 (only if S2 says it's needed).
2.7a Favorites (`FV:2`, incl. `r:resMD` extraction). 2.7b Playlists (`SQ:`). 2.7c Queue (`Q:0`).
2.8a Replace-and-play: container/track path. 2.8b Replace-and-play: stream path.
2.9a Enqueue at end. 2.9b Enqueue as next (`EnqueueAsNext=1` + `DesiredFirstTrackNumberEnqueued`). 2.9c A SOAP fault on enqueue becomes a typed "not enqueueable" error (per S2's recorded fault).
2.10 Queue jump: `SetAVTransportURI(x-rincon-queue:<uid>#0)` then `Seek TRACK_NR`; call order asserted.
2.11a Play-mode enum encode/decode (all six values; `SHUFFLE` == shuffle+repeat-all asserted) + read via `GetTransportSettings`. 2.11b Shuffle/repeat toggle semantics over the enum.
2.12 Sleep timer: set, cancel (empty string), read back remaining.

### Phase 3 — `state/`
3.1 Cache read/write: **atomic** (temp file + rename), TTL with injected clock; stale or corrupt = miss.
3.2 Active group persisted by player UUID.
3.3 Target resolution per the rule above. Tests: stored UUID is a non-coordinator member; stored UUID absent; nothing stored; nothing playing.
3.4 Single-flight refresh lock, including stale-lock recovery.

### Phase 4 — `hub/` (pure functions)
4.0 Neutral item model + the `hub` → `do` argument encoding; round-trip test.
4.1 Now-playing row (Enter play/pause; ⌘ next; ⌥ previous).
4.2 Top-level menu ordering.
4.3 Fuzzy filtering across sections.
4.4 Favorite/Playlist items with Enter/⌘/⌥ mods; degraded ⌘/⌥ per S2.
4.5 Queue items, current track marked.
4.6 Room items.
4.7 `vol 35` parsing (clamp 0–100; garbage → no item) and ± step items.
4.8 Shuffle/repeat items reflecting current mode; sleep-timer items reflecting remaining time.
4.9 Single renderer to Script Filter JSON (golden file); `rerun` only when a refresh is pending; "Loading Sonos…" item on cold cache.

### Phase 5 — Wiring
5.1 `filter` end to end from a warm cache (golden JSON). **Test: a fake that fails on any request proves `filter` makes zero network calls.**
5.2 `refresh` subcommand: talks to the speaker, writes cache atomically, records success/unreachable; `filter` spawns it detached when data is stale.
5.3 `do` subcommand: action → sonos call on the coordinator; invalidates affected cache entries; failure → error path per S3.
5.4 Speaker unreachable: `filter` shows one "Can't reach Sonos" item based on the recorded refresh outcome; never crashes.

### Phase 6 — Packaging + manual acceptance
6.1a arm64 build (+ `codesign -f -s -` only if S3 shows it's needed).
6.1b `info.plist` templated from your exported skeleton.
6.1c Zip to `.alfredworkflow`; test asserts archive contents.
6.2 You import it and walk the checklist below. Not automatable, and I'll say so rather than claim it passes.

## Verification
- `go vet ./... && go test ./...` green.
- **Latency:** `time ./sonos-alfred filter ''` with a warm cache, ≥5 runs, report p50/p95. This measures the binary only; Alfred adds process-spawn overhead, and the first run after each rebuild is slower.
- **Unreachable speaker:** set the config IP override to `192.0.2.1` (TEST-NET-1, unroutable). Expect one "Can't reach Sonos" item, no crash.
- **Sleep timer:** set 15 min, assert `GetRemainingSleepTimerDuration` is non-empty; cancel, assert empty.
- **Manual in Alfred:** play/pause; volume set and ± step; favorite replace/⌘/⌥ (including one radio favorite); queue jump **while a radio favorite is playing**; room switch; shuffle/repeat; error notification.

## Changes from review
- Fixed my API errors: `EnqueueAsNext` is an argument; play mode is a six-value enum; queue ID is `Q:0`; queue jump needs a preceding `SetAVTransportURI`; two topology shapes; Gatekeeper quarantine doesn't apply to locally built workflows.
- Resolved the filtering contradiction: per-keystroke Go filtering, cache-only reads, background refresh (your decision).
- Dropped AwGo as output layer and dropped amd64 (your decisions).
- Target resolution now follows regrouping. Cache writes are atomic. Refresh is single-flight.
- Fake speaker is strict; escaping and call order are tested.
- Oversized increments split; item model and arg encoding defined before the menu increments.
- Verification made measurable. Notification mechanism turned into a spike with an `osascript` fallback.
- Not adopted as written: the reviewer's unbounded `rerun: 0.3` (now bounded and only while a refresh is pending), and the Alfred 5.5+ version question (moot with `cache` unused).

## References
- svrooij Sonos services: https://sonos.svrooij.io/services/
- SoCo `core.py`, `zonegroupstate.py`, `data_structures.py`: https://github.com/SoCo/SoCo
- sonoscli (Go, MIT): https://github.com/steipete/sonoscli
- Alfred Script Filter JSON: https://www.alfredapp.com/help/workflows/inputs/script-filter/json/
