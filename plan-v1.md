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
| Volume | Type a number (`son vol 35`) + one "Volume N" row (Enter steps up, ⌥ steps down; fixed step of 5, see Amendments). Group volume default |
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
- One cached state (topology, now-playing, volume, lists, play mode, sleep timer) is fresh for 10 s; after that `filter` shows it and spawns a background refresh, and `do` expires it (see Amendments). Replaces the per-entry TTLs first proposed here.
- Optional "speaker IP" in workflow config as a discovery fallback. `/24` subnet scan deferred (no VPN).
- Bundle ID `org.infernus.sonos-alfred`; assumes **Alfred 5 + Powerpack** (exact minor version doesn't matter, since `cache` is unused).
- Sleep timer: presets (15/30/60 min, off).

## Unverified → resolved by spikes before feature work
1. Local UPnP still works on your speakers/firmware (one community post claims gradual S2 deprecation; unconfirmed). **Resolved by S1: works.**
2. Whether radio favorites can be enqueued. SoCo imposes no such restriction; reports of fault 800 exist. Record the exact fault.
3. Whether `SnapshotGroupVolume` must be called explicitly. **Resolved by S2: not required.**
4. Stereo-pair and soundbar+sub topology shape on your firmware.
5. How Alfred surfaces a Run Script failure. The Alfred docs are silent on exit status/stderr. **Resolved by S3: stdout feeds the notification; stderr alone is invisible.**
6. Whether Go's arm64 output is already ad-hoc signed and runs cleanly from an imported workflow. **Resolved by S3: yes.**

## Spike findings

### S1 — discovery + topology (2026-09-20)
- **Unverified #1 resolved: local UPnP works.** `GetZoneGroupState` on port 1400 returned HTTP 200 from all 9 responding IPs (firmware 97.1-80312). No need to change backend.
- **Sandbox:** SSDP multicast finds nothing from inside the nono sandbox (run outside it: 9 speakers). Unicast HTTP to a known IP works (25 ms). So SSDP can only be hand-verified outside the sandbox; everything else can be tested here against a known IP.
- **6 rooms, 9 IPs.** Rooms: Kitchen, Living Room, Garden Room, Office, Dining Room, Play. The 3 extra SSDP responders are the Living Room's rear surrounds and sub. **Discovery results must be deduplicated through the topology, never used as a room list.**
- **Unverified #4 partly resolved.** Soundbar+sub+surrounds is the nested shape: `<Satellite Invisible="1">` children inside the soundbar's `ZoneGroupMember` (rears carry the room's `ZoneName`, the sub is named `Sub`). The soundbar member also has `HTSatChanMapSet`. **No stereo pair exists in this household**, so the top-level `Invisible="1"` shape is not captured; the 2.2b fixture will be synthetic, built from SoCo/svrooij docs and labelled as such.
- **Group order differs per responding speaker** (same groups, shuffled). Sort rooms by name for a stable UI.
- **Group `ID` prefix is not the coordinator**: Garden Room's coordinator is `RINCON_3333…` but its group ID starts `RINCON_1111…`. Always use the `Coordinator` attribute, never parse the ID.
- All 6 groups are currently single-member. Multi-member groups are untested against real hardware.
- Responses from different speakers differ only in non-structural attributes (e.g. `LineInActiveMask`).
- Fixture: `testdata/zonegroupstate-home-theatre.xml` (from the soundbar, since sanitised: fake speaker IDs, `192.0.2.x` addresses).
- SSDP timing not yet recorded (S1 output not seen by me).

### S2 (read-only part) — Browse (2026-09-20)
Nothing was played or changed; live playback/enqueue/volume tests still need your go-ahead (see "Open decisions").
- `Browse` works for `FV:2`, `SQ:` and `Q:0` (~50 ms each; the first `FV:2` call took 4.1 s, so favorites refresh must stay in the background).
- **Favorites: only 2, both Sonos Radio "shortcuts" with an EMPTY `<res>`** (`<r:type>shortcut</r:type>`; "Discover Sonos Radio", "Trending Now"). Their playable identity is only in `r:resMD` (an `object.container` whose `<desc>` token `SA_RINCON77575_X_#Svc77575-0-Token` implies Sonos Radio, service id 303 = (77575-7)/256). **The plan's "play the `<res>` URI with `r:resMD`" does not apply to these.** How to play one is unknown until a live test. Do not assume a `x-sonosapi-radio:` URI form.
- **Playlists (`SQ:`): empty.** Apple Music playlists are not Sonos playlists, so the Playlists section would be empty for this household.
- **Queue (`Q:0`): 46 Apple Music tracks** (`x-sonos-http:song%3a…?sid=204&flags=8232&sn=5`). Queue rendering and queue jump are testable as planned.
- `resMD` is double-escaped inside `Browse`'s `Result`, as expected; the `Trending Now` `resMD` has an empty `dc:title` and an `id`/`parentID` that differ in case, so favorite titles must come from the outer item, not `resMD`.
- Fixtures: `testdata/browse-favorites-sonos-radio-shortcuts.xml`, `browse-playlists-empty.xml`, `browse-queue-apple-music.xml` (since sanitised: the queue fixture now holds made-up titles).

### S2 (live, Dining Room) — playback, queue, volume (2026-09-20)
Run on Dining Room (`192.0.2.29`) with your consent; volume stayed 12–14 and was restored (also queue source, play mode, sleep timer). Its queue was replaced (1 → 151 tracks). You added two Apple Music album favorites; `FV:2` now has 4 items.
- **Favorite shapes.** Apple Music albums are `r:type=instantPlay` with a real `<res>` (`x-rincon-cpcontainer:…?sid=204&flags=8300&sn=5`) plus `resMD` (an `object.container.album.musicAlbum`). Sonos Radio items are `r:type=shortcut` with an empty `<res>`.
- **2.8a Replace-and-play (container) CONFIRMED:** `RemoveAllTracksFromQueue` → `AddURIToQueue(res, resMD, 0, 0)` (returned `FirstTrackNumberEnqueued=1, NumTracksAdded=12`) → `SetAVTransportURI(x-rincon-queue:<uid>#0)` → `Play`; playing within ~2 s. `AddURIToQueue` takes ~0.6 s for an album.
- **2.9a Enqueue at end CONFIRMED:** `EnqueueAsNext=0, DesiredFirstTrackNumberEnqueued=0` appends (`FirstTrackNumberEnqueued=13` on a 12-track queue).
- **2.9b Play next needs the current track number:** `EnqueueAsNext=1` + `DesiredFirstTrackNumberEnqueued=<current+1>` lands right after the current track (returned 2). **`EnqueueAsNext=1` with desired `0` APPENDED at the end (105), not next.** So `do` must read the current track (`GetPositionInfo`) first. Behaviour when the group is on a stream (no queue position) is undecided: fall back to enqueue at end.
- **2.10 Queue jump CONFIRMED while on a stream:** a bare `Seek TRACK_NR` fails with HTTP 500, UPnP errorCode **701**; `SetAVTransportURI(x-rincon-queue:<uid>#0)` → `Seek TRACK_NR=3` → `Play` landed on track 3.
- **Radio enqueue (partial):** enqueuing a raw `x-rincon-mp3radio://…` URI with empty metadata **succeeded** (no fault). This is only a proxy: you have no playable Sonos Radio favorite, so the "radio favorite can't be enqueued (fault 800)" question is still unanswered for real service-based radio. The ⌘/⌥ degradation in 4.4 stays "decided from S2", and S2 says: allow it for URI-based streams, undecided for others.
- **Sonos Radio shortcuts are NOT playable via UPnP as tried:** (1) `SetAVTransportURI` with an empty URI + `resMD` returns 200 but leaves the group with no media (`NrTracks 0`, `Play` → 701); (2) a guessed `x-sonosapi-radio:sd%3aUK%3atrending-now?sid=303&flags=8300&sn=0` via `AddURIToQueue` → errorCode **800** (a wrong guess proves nothing). **Decision needed: treat `shortcut` favorites as not playable in v1** (hide, or show as invalid), unless you want a further spike.
- **Play mode CONFIRMED:** `SetPlayMode SHUFFLE` reads back `SHUFFLE`; `SHUFFLE_NOREPEAT` and `NORMAL` also round-trip via `GetTransportSettings`.
- **Sleep timer CONFIRMED:** `ConfigureSleepTimer 00:15:00` → `GetRemainingSleepTimerDuration` `00:15:00` (generation 1); `ConfigureSleepTimer ""` → remaining `""` (generation 0).
- **Group volume (single-member group):** `SetRelativeGroupVolume(+2)` → `NewVolume 14`; `(-2)` → `12`; `SetGroupVolume 12` OK; group and member volume agree.
- **Playlists:** `SQ:` is empty here. The Playlists section will be empty for this household.
- Fixtures: `testdata/browse-favorites-albums-and-shortcuts.xml` (replaces the earlier favorites fixture) and `testdata/s2-dining-room-exchanges.jsonl` (the exact requests the speaker accepted, plus the 701 fault; the spike escapes values with Go's `html.EscapeString`, so `'` becomes `&#39;`).
- **Group volume with two members (Dining Room coordinator, Office member; 12/1, group reads 6) — unverified #3 resolved: a snapshot is NOT required.** `SetGroupVolume` and `SetRelativeGroupVolume` work without one: the group value lands on the target and `NewVolume` matches the readback. So **2.6c is dropped**. Caveats: (1) per-member changes are **not a simple ratio** (12/1 → `SetGroupVolume 8` gave 13/3 without a snapshot and 12/4 with one; `-5` gave 6/2, with Office *rising*, which I suspect but have not confirmed is a stale internal snapshot from the previous call); (2) `GetGroupVolume` is not a pure function of member volumes (members restored to 12/1 read group 7, 8, 9, 7 across runs), so a group volume read right after a member change can be stale. Fine for single-room use; treat multi-room group volume as best-effort.
- **Non-coordinator commands fail:** `GetGroupVolume` sent to Office (a member) returned errorCode **701**. This confirms the "always target the current coordinator" rule.
- **Still open:** SSDP timing (S1 output not seen).

### S3 — packaging + Alfred behaviour (2026-09-20)
Imported `dist/S3-spike.alfredworkflow` (arm64 Go binary, inline scripts) and ran every path with the debugger open.
- **Unverified #6 resolved:** Go's arm64 output is `adhoc, linker-signed` by default and ran from the imported workflow with no extra `codesign` and no prompt reported. **6.1a needs no `codesign` step.**
- **Script nodes:** inline script `./sonos-alfred filter "$1"` / `./sonos-alfred do "$1"` with plist `type 11` and `scriptargtype 1` works for both Script Filter and Run Script, and the binary is found in the workflow directory. `type 8` (external script, used by the skeleton) is not needed, so unverified #5's plist question is moot.
- **`mods` work end to end:** ⌘ and ⌥ passed their own `arg` (`ok-cmd`, `ok-alt`) to the Run Script. (Whether the subtitle changes while a modifier is held was not observed.)
- **Error mechanism (unverified #5 resolved):** the Run Script's **stdout** feeds Post Notification (`text = {query}`, `onlyshowifquerypopulated = true`) and the notification is posted **even if the script exits non-zero**. Empty stdout posts nothing, so success is silent by construction.
  - stdout text + exit 0 → notification. stdout text + exit 1 → notification (and a debugger `ERROR:` line).
  - **stderr only + exit 1 → nothing user-visible** (only a debugger `ERROR:` line). So **every failure path in `do` must print its message to stdout**, including panics (recover in `main`).
  - The `osascript` fallback works but is not needed: dropped.
- **`{query}` in result text is substituted by Alfred** (an item subtitle written "via {query}" rendered as "via fail-text"). Never put a literal `{query}` in titles or subtitles.
- **Latency baseline:** with a trivial Go binary, Alfred's "Queuing argument" → "Script finished" took ~30 ms warm (~120 ms for the first run after import). A per-keystroke script has that floor before any real work.
- Rendering: 5 items rendered with the expected titles/subtitles; ⌘1–⌘5 quick-select hints appear on the right (Alfred's default).

### Phase 1 gate: PASSED (2026-09-20)
UPnP is alive; no backend change. Adjustments to the phases below, all from the findings above:
- **2.6c dropped** (snapshot not required).
- **Favorites:** `r:type=shortcut` favorites (empty `<res>`, Sonos Radio) are **not playable via UPnP as tried and are hidden in v1**; `instantPlay` favorites (Apple Music albums) use the container path 2.8a. 2.8b (stream path) is exercised only with URI-based streams.
- **2.9b** needs the current track number (`GetPositionInfo`) for "play next"; with no queue position (stream) it falls back to enqueue-at-end.
- **2.9c** "not enqueueable" typed error is kept, keyed on a SOAP fault on `AddURIToQueue`; the exact fault for a real radio favorite remains unrecorded (no playable one exists here).
- **2.2b** stereo-pair fixture is synthetic (none in this household).
- **Playlists (`SQ:`)** are empty here; the section renders only when non-empty.
- **`do` failure path** prints to stdout (see S3).
- SSDP timing was never captured by me and is not blocking.
- Spike code deleted; fixtures and findings are the outputs.

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

## Amendments after implementation (2026-09-21)
Where the built workflow differs from, or settles, what is written above.
- **Cache:** a single `state` entry, fresh for 10 s, not per-entry TTLs (you confirmed this in Phase 5).
- **Volume:** one "Volume N" row: Enter steps up 5, ⌥ steps down 5. The step is fixed, not configurable. Typing `vol 35` still offers "Set volume 35".
- **Rerun cap:** at most 100 reruns (about 30 s at Alfred's 0.3 s rerun). The count lives in the cache directory, because each `filter` run is a new process. A pause of over 10 s starts a fresh count. If the count cannot be saved, `filter` does not ask for a rerun.
- **Ranking:** rows whose title contains the query come before rows that only have its letters scattered through the title.
- **Playlists:** the speaker returns Sonos playlists as `<container>` elements, not `<item>`, so the first parser saw none. Fixed from a real capture (`testdata/browse-playlists-one.xml`). Enqueuing a playlist with empty metadata was verified on Dining Room: the speaker replaced the queue with the playlist's tracks. Playing it from Alfred is not yet verified. The "Playlists empty" notes in S1 and S2 were true of that day's household.
- **Key hints:** favorite and playlist rows say what each key does ("↩ play · ⌘↩ add to end · ⌥↩ play next"). The volume row already had one. The now-playing row's ⌘ next and ⌥ previous have no hint yet.
- **`{query}` guard:** S3 saw Alfred substitute `{query}` in result text. The renderer splits a literal `{query}` in titles and subtitles with an invisible character, so a track with that name reads the same. Encoded actions already escape the braces.
- **Speaker IP:** the `SONOS_HOST` workflow setting ("Speaker IP") overrides SSDP discovery; empty means discover. The Alfred configuration format for it is unverified.
- **Fixtures:** the recorded fixtures are sanitised before they are committed (see `AGENTS.md`), so the raw-capture notes in S1 and S2 above describe what the speakers returned, not what the files now hold.
- **Packaging:** the checked-in `workflow/info.plist` plus `scripts/package.sh <out>`. The archive holds the arm64 binary and the plist only, with no icon.

## References
- svrooij Sonos services: https://sonos.svrooij.io/services/
- SoCo `core.py`, `zonegroupstate.py`, `data_structures.py`: https://github.com/SoCo/SoCo
- sonoscli (Go, MIT): https://github.com/steipete/sonoscli
- Alfred Script Filter JSON: https://www.alfredapp.com/help/workflows/inputs/script-filter/json/
