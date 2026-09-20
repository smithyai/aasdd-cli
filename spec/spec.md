## AASDD CLI

**AASDD:** v2
**Version:** 0.3.0

A command-line tool for verifying, scaffolding, exporting, importing, diffing, and graphing AASDD specs.

### Purpose

For anyone who writes or consumes AASDD specs and needs the methodology enforced mechanically: authors checking a spec as they write it, agents verifying their own edits, and continuous integration gating a merge. It turns the structural rules and readiness conditions of the methodology into a single command whose verdict can be trusted without rereading the methodology documents.

### Non-Goals

- Does not implement specs or generate code from them.
- Does not judge whether an invariant is well chosen; it checks structure, references, and coverage, not meaning.
- Does not modify a spec in place; every command that produces files writes to a new location.

### Success Criteria

| Criterion                                                                                                            | Abilities          | Scenarios                                                            |
| -------------------------------------------------------------------------------------------------------------------- | ------------------ | -------------------------------------------------------------------- |
| A spec that violates a structural rule or a readiness condition is reported with the rule, the file, and the reason. | `Verify`           | —                                                                    |
| A conformant spec is reported as conformant, and a condition that applies only at 1.0.0 is not reported for a draft. | `Verify`           | `scaffolds-and-verifies`, `scaffolds-example-and-verifies`           |
| A new spec directory can be created that already conforms.                                                           | `Scaffold`         | `scaffolds-and-verifies`, `scaffolds-example-and-verifies`           |
| A spec can be exported to a structured file and reconstructed from it without losing any construct.                  | `Export`, `Import` | `exports-imports-and-verifies`, `exports-imports-verifies-and-diffs` |
| Two specs can be compared and every structural difference between them listed.                                       | `Diff`             | `diffs-two-scaffolds`, `exports-imports-verifies-and-diffs`          |
| The ability tree and type usage of a spec can be rendered as a graph in a standard format.                           | `Graph`            | `scaffolds-and-graphs`, `graphs-in-dot-format`                       |
| The AASDD versions the tool understands can be listed.                                                               | `ListVersions`     | —                                                                    |

### Invariants

- The tool never overwrites or modifies any existing file.
- All output is written to stdout; all errors and diagnostics are written to stderr.
- Exit code is `0` when the ability completes without error, non-zero otherwise.
