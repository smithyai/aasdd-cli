## Scaffolds Example and Verifies

Scaffolding a worked-example spec directory and immediately verifying it produces a conformant result.

> `Scaffold` → `Verify`

- `Scaffold` is invoked with `example` set to `true`
- `Scaffold.result.files_created` is non-empty
- Every path in `Scaffold.result.files_created` exists on disk after `Scaffold` completes
- `Verify.result.passed` is `true` when run against the scaffolded directory
