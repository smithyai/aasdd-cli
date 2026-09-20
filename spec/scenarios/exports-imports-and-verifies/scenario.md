## Exports Imports and Verifies

Exporting a spec directory and importing the result produces a directory that is structurally equivalent to the original.

> `Export` → `Import` → `Verify`

- `Export.result.file_count` equals `Import.result.file_count`
- Every spec file from the original directory exists at the same relative path in the imported directory
- `Verify.result.passed` is `true` when run against the imported directory
