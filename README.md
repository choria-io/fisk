# Overview

Fisk is a [fluent-style](http://en.wikipedia.org/wiki/Fluent_interface), type-safe command-line parser. It supports flags, nested commands, and positional arguments.

This is a fork of [kingpin](https://github.com/alecthomas/kingpin), a very nice CLI framework that has been in limbo for a few years. As this project and others we 
work on are heavily invested in Kingpin we thought to revive it for our needs. We'll likely make some breaking changes, so kingpin is kind of serving as a starting
point here rather than this being designed as a direct continuation of that project.

For full help and intro see [kingpin](https://github.com/alecthomas/kingpin), this README will focus on our local changes.

## Versions

We are not continuing the versioning scheme of Kingpin, the Go community has introduced onerous SemVer restrictions, we will start from 0.0.1 and probably never pass 0.x.x.

Some historical points in time are kept:

| Tag    | Description                                                     |
|--------|-----------------------------------------------------------------|
| v0.0.1 | Corresponds to the v2.2.6 release of Kingpin                    |
| v0.0.2 | Corresponds with the master of Kingpin at the time of this fork |
| v0.1.0 | The first release under `choria-io` org                         |

## Notable Changes

 * Renamed `master` branch to `main`
 * Incorporate `github.com/alecthomas/units` and `github.com/alecthomas/template` as local packages
 * Changes to make `staticcheck` happy
 * A new default template that shortens the help on large apps, old default preserved as `KingpinDefaultUsageTemplate`

## Cheats

I really like [cheat](https://github.com/cheat/cheat), a great little tool that gives access to bite-sized hints on what's great about a CLI tool.

Fisk supports cheats natively, you can get cheat formatted hints right from the app with no extra dependencies or export cheats into the `cheat` app for use via its interface and integrations.

```nohighlight
$ nats cheat pub
# To publish 100 messages with a random body between 100 and 1000 characters
nats pub destination.subject "{{ Random 100 1000 }}" -H Count:{{ Count }} --count 100
```

Let's look how that is done:

```go
// WithCheats() enables cheats without adding any to the top, you
// can also just call Cheat() at the top to both set a cheat and enable it
// once enabled at the top all cheats in all sub commands are accessible
nats := fisk.New("nats", "NATS Utility").WithCheats()

pub := nats.Command("pub", "Publish utility")
pub.Cheat(`# To publish 100 messages with a random.....`)
```

After that your app will have a new command `cheat` that gives access to the cheats. It will show
a list of cheats when trying to access a command without cheats or when running `nats cheat --list`.

```nohighlight
$ nats cheat unknown
Available Cheats:

  nats/pub
```
