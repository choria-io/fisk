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
 * Integration with [cheat](https://github.com/cheat/cheat) (see [below](#cheats))
 * Unnegatable booleans using a new `UnNegatableBool()` flag type, backwards compatibility kept
 * Extended parsing for durations that include weeks (`w`, `W`), months (`M`), years (`y`, `Y`) and days (`d`, `D`) units (`v0.1.3` or newer)
 * More contextually useful help when using `app.MustParseWithUsage(os.Args[1:])` (`v0.1.4` or newer)
 * Default usage template is `CompactMainUsageTemplate` since `v0.3.0` 

### UnNegatableBool

Fisk will add to all `Bool()` kind flags a negated version, in other words `--force` will also get `--no-force` added
and the usage will show these negatable booleans.

Often though one does not want to have the negatable version of a boolean added, with fisk you can achieve this using
our `UnNegatableBool()` which would just be the basic boolean flag with no negatable version.

Arguably `Bool()` should be un-negatable and we should have added a `NagatableBool()` but the decision was made to keep
existing apps backward compatible.

### Cheats

I really like [cheat](https://github.com/cheat/cheat), a great little tool that gives access to bite-sized hints on what's great about a CLI tool.

Since `v0.1.1` Fisk supports cheats natively, you can get cheat formatted hints right from the app with no extra dependencies or export cheats into the `cheat` app for use via its interface and integrations.

```nohighlight
$ nats cheat pub
# To publish 100 messages with a random body between 100 and 1000 characters
nats pub destination.subject "{{ Random 100 1000 }}" -H Count:{{ Count }} --count 100
```

Cheats are stored in a `map[string]string`, meaning it's flat, does not support subs and when saving cheats, due to the
nature of the fluent api, 2 cheats with the same name will overwrite each other.

I therefore suggest you place your cheat in the top command for an intro and then place them where you need them in the 
first sub command only not deeper, this makes it easy to avoid clashes and easy for your users to discover them.

Let's look how that is done:

```go
// WithCheats() enables cheats without adding any to the top, you
// can also just call Cheat() at the top to both set a cheat and enable it
// once enabled at the top all cheats in all sub commands are accessible
//
// Cheats can have multiple tags, here we set the tags "middleware", and "nats"
// that will be used when saving the cheats.  If no tags are supplied the
// application name is used as the only tag
nats := fisk.New("nats", "NATS Utility").WithCheats("middleware", "nats")

pub := nats.Command("pub", "Publish utility")
// The cheat will be available as "pub", the if the first argument
// is empty the name of the command will be used
pub.Cheat("pub", `# To publish 100 messages with a random.....`)
```

After that your app will have a new command `cheat` that gives access to the cheats. It will show
a list of cheats when trying to access a command without cheats or when running `nats cheat --list`.

```nohighlight
$ nats cheat unknown
Available Cheats:

  pub
```

You can save your cheats to a directory of your choice with `nats cheat --save /some/dir`, the directory
must not already exist.

## External Plugins

Often one wants to make a CLI tool that can be extended using plugins.  Think for example the `nats` CLI that is built
using `fisk`, it may want to add some commercial offering-specific commands that appear in the `nats` command as a fully
native built-in command.

**NOTE:** This is an experimental feature that will see some iteration in the short term future.

Fisk `0.5.0` supports extending itself at run-time in this manner and any application built using this version of fisk
can be plugged into another fisk application.

The host application need to do some work but the ones being embedded will just work.

```go
app := fisk.New("nats", "NATS Utility")

// now load your plugin models from disk and extend the command
model := loadModelJson("ngs")

app.ExternalPluginCommand("/opt/nats/bin/ngs", model)
```

The `model` above can be obtained by running a `fisk` built command with `--fisk-introspect` flag.

The idea is that you would detect and register new plugins into your tool, you would call them once with the introspect
flag and cache that model.  All future startups would embed this command - here `nats ngs` - right into the command flow.

The model is our [ApplicationModel](https://pkg.go.dev/github.com/choria-io/fisk#ApplicationModel). Plugins written in
other languages or using other CLI frameworks would need to emit a compatible model.

Care should be taken not to have clashes with the top level global flags of the app you are embedding into, but if you
do have a clash the value will be passed from top level into your app invocation.  This should be good enough for most
cases but could leed to some unexpected results when you might have a different concept of what a flag means than the one
you are embedded into.