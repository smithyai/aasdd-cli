## DiffsTwoScaffolds

Scaffolding a minimal spec and an example spec then diffing them reveals the additional constructs in the example.

> `Scaffold` → `Scaffold` → `Diff`

- The first `Scaffold` is invoked with `example` set to `false`
- The second `Scaffold` is invoked with `example` set to `true`
- `Diff.result.changed` is `true`
- `Diff.result.entries` contains at least one `Added` entry for each of: `ability`, `concept`, `decision`, `scenario`
