## Exports Imports Verifies and Diffs

Exporting a spec directory, importing the result, verifying the import, and diffing the original against the import confirms both conformance and structural equivalence.

> `Export` → `Import` → `Verify` → `Diff`

- `Export.result.file_count` equals `Import.result.file_count`
- `Verify.result.passed` is `true` when run against the imported directory
- `Diff.result.changed` is `false`
- `Diff.result.entries` is empty
