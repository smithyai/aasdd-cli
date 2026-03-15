## AASDD CLI

**AASDD:** v1
**Version:** 0.2.1

A command-line tool for verifying, scaffolding, exporting, importing, diffing, and graphing AASDD specs.

### Invariants

- The tool never overwrites or modifies any existing file.
- All output is written to stdout; all errors and diagnostics are written to stderr.
- Exit code is `0` when the ability completes without error, non-zero otherwise.
