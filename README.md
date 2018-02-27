# isolator

Lightweight [process isolation](https://en.wikipedia.org/wiki/Process_isolation) for Windows + macOS + Linux extracted from [Itch Butler](https://github.com/itchio/butler).

Isolates the process in its directory, and jails it from interacting with other processes while allowing things like network. This should provide basic protection against the process stealing data or breaking things outside its directory (such as a game or web site). However, this does not provide protection from things like kernel bugs or misconfigured permissions on the filesystem.

Think Docker, except one command, single binary, no containers, and works on all platforms with no install. May require elevated privs on first run (only), which it will automatically ask for.

Used isolating browser process in [Exokit](https://github.com/modulesio/exokit), but it should work for anything.

## Usage

```
go build
./isolator /name/of/exe # run exe in container, in CWD
```
