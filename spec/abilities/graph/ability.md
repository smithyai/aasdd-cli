## Graph

Generates a dependency graph of a spec directory showing how abilities, concepts, decisions, and scenarios relate to each other.

### Inputs

| Name     | Type                                                       | Description                               |
| -------- | ---------------------------------------------------------- | ----------------------------------------- |
| `target` | [SpecTarget](../../concepts/cli/concept.md#spectarget)     | The spec directory to graph.              |
| `format` | [GraphFormat](../../concepts/graph/concept.md#graphformat) | The output format. Defaults to `Mermaid`. |

### Outputs

| Name     | Type                                                       | Description         |
| -------- | ---------------------------------------------------------- | ------------------- |
| `result` | [GraphResult](../../concepts/graph/concept.md#graphresult) | The rendered graph. |

### Invariants

- Every ability, concept, decision, and scenario in the spec directory is represented as a node.
- Sub-abilities are connected to their parent ability with a `sub-ability` edge.
- When an ability's inputs or outputs reference a concept type, the ability is connected to that concept with an `input` or `output` edge.
- When a scenario's trace references an ability, the scenario is connected to that ability with a `traces` edge.
- `result.content` is valid source in the requested format — parseable by Mermaid or Graphviz respectively.
- `result.node_count` equals the number of distinct nodes in the graph.
- `result.edge_count` equals the number of distinct edges in the graph.

### Failure Modes

| Failure          | Condition                                 | Effect                                       |
| ---------------- | ----------------------------------------- | -------------------------------------------- |
| `TargetNotFound` | `target.path` does not exist.             | Error written to stderr; exit code non-zero. |
| `TargetIsFile`   | `target.path` is a file, not a directory. | Error written to stderr; exit code non-zero. |
| `ReadError`      | A file under `target` cannot be read.     | Error written to stderr; exit code non-zero. |

### Notes

The graph is written to stdout. `--verbose` has no effect. `--progress` has no effect.
