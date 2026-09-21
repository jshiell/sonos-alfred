# Sonos for Alfred

An [Alfred](https://www.alfredapp.com/) 5 workflow that controls Sonos speakers from one keyword, `son`. It talks to the speakers directly over the local network (UPnP/SOAP on port 1400), so there is no cloud account, no Sonos app and no login.

It is built for one household. It has been checked against that household's speakers and nothing else.

## What it does

Type `son` and pick a row. Type more to filter: rows whose title contains what you typed come first, then rows that have your letters in order.

| Row | ↩ | ⌘↩ | ⌥↩ |
|---|---|---|---|
| Now playing | play / pause | next track | previous track |
| Volume | up 5 | | down 5 |
| Favorite or Sonos playlist | replace the queue and play | add to the end | play next |
| Queue track | jump to it | | |
| Room | control that room's group | | |
| Shuffle, Repeat | toggle shuffle / cycle repeat off, all, one | | |
| Sleep in 15 / 30 / 60 minutes | start the timer | | |
| Cancel sleep timer (only while one runs) | cancel it | | |

`son vol 35` adds a "Set volume 35" row. Volume is the group's volume.

Things worth knowing:

- **Which group.** Commands go to the group's coordinator. With no room chosen, that's whichever group is playing; picking a room makes its group the target until you pick another.
- **Streams.** Sonos refuses shuffle and repeat while a radio stream or AirPlay is playing, so those rows are hidden then. When a source gives no title, such as AirPlay, the now-playing row says "Playing".
- **Success is silent.** A failed action shows a notification saying why.

Not included: search, grouping and scenes, line-in, text to speech, Intel Macs.

## Install

You need an Apple silicon Mac, Alfred 5 with the Powerpack, and [mise](https://mise.jdx.dev/) (it installs the pinned Go version).

```
scripts/package.sh dist/Sonos.alfredworkflow
```

Open `dist/Sonos.alfredworkflow` to import it. The binary isn't signed or notarised, because you build it yourself.

By default the workflow finds a speaker by SSDP discovery. To skip discovery, set **Speaker IP** in the workflow's configuration to any speaker's address, for example `192.168.1.20`.

## How it works

`son` runs `sonos-alfred filter`, which only reads a cache and prints Alfred's Script Filter JSON, so it answers straight away and never touches the network. When the cache is older than 10 seconds, it starts a detached `sonos-alfred refresh`, which reads the household from the speakers and rewrites the cache, and asks Alfred to rerun the filter until that lands (at most 100 reruns). Choosing a row runs `sonos-alfred do`, which acts on the speakers and prints the reason if it fails, which Alfred shows as a notification.

## Troubleshooting

- **"Can't reach Sonos".** The subtitle says why the last refresh failed. Check the speakers are on and on the same network, or set **Speaker IP**.
- **"… context deadline exceeded" right after installing or rebuilding.** A freshly built binary can time out on its first network calls for a minute or two, then run normally. Wait and try again. The cause isn't known.

## Develop

Go is pinned in `mise.toml`. The gate is:

```
go vet ./... && go test ./...
```

| Package | Job |
|---|---|
| `sonos` | UPnP/SOAP client: discovery, topology, transport, volume, browse, queue, play mode, sleep timer |
| `hub` | What the rows are and what each key does, from the cached state |
| `alfredjson` | Renders rows as Script Filter JSON |
| `state` | Cache, chosen room, target resolution, refresh lock |
| `app` | The `filter`, `refresh` and `do` commands |
| `workflow` | `info.plist` and its packaging test |
| `internal/fakespeaker` | Strict fake speaker used by the tests |

Tests use the fake speaker and recorded fixtures in `testdata/`. The fixtures are sanitised captures of a real household (addresses are `192.0.2.x`, speaker IDs are `RINCON_1111…`, titles are made up); sanitise any new capture the same way before committing it.

`plan-v1.md` has the design, the spike findings and what changed after implementation. `AGENTS.md` has the working notes for coding agents, including known issues.
