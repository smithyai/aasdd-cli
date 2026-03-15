## Graph

Generates a dependency graph of a spec directory showing how abilities relate to each other and to the concepts they consume and produce.

### Inputs

| Name      | Type                                                                     | Description                                                                                                                                                                                                               |
| --------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `target`  | [SpecTarget](../../concepts/cli/concept.md#spectarget)                   | The spec directory to graph.                                                                                                                                                                                              |
| `format`  | [GraphFormat](../../concepts/graph/concept.md#graphformat)               | The output format. Defaults to `Mermaid`.                                                                                                                                                                                 |
| `include` | optional `scenarios` \| `decisions` \| `state-machine` (comma-separated) | When set, adds the specified supplementary node kinds to the default ability-anchored graph. Abilities and concepts are always included. Omitted means abilities and concepts only.                                       |
| `depth`   | optional integer                                                         | When set to a positive integer, limits the ability tree walk to that many levels deep. `1` shows only root abilities; `2` shows root abilities and their direct sub-abilities; and so on. `0` or omitted means unlimited. |
| `root`    | optional string                                                          | When set, re-roots the graph at the ability whose last path segment matches this value (case-insensitive). Only that ability and its descendants are included. Omitted means the spec root.                               |

### Outputs

| Name     | Type                                                       | Description         |
| -------- | ---------------------------------------------------------- | ------------------- |
| `result` | [GraphResult](../../concepts/graph/concept.md#graphresult) | The rendered graph. |

### Invariants

- Ability and concept nodes are always present in the output.
- Sub-abilities are connected to their parent ability with a `sub-ability` edge.
- When an ability's inputs or outputs reference a concept type, the ability is connected to that concept with an `input` or `output` edge.
- When `include` contains `scenarios`, scenario nodes are added and each scenario is connected to the abilities it traces with a `traces` edge.
- When `include` contains `decisions`, decision nodes are added with no edges.
- When `include` contains `state-machine` and the spec has a state machine, the state machine node is added and connected to each ability it orchestrates with an `orchestrates` edge.
- When `root` is set, only the matching ability, its descendants, and concepts connected to them appear in the output.
- When `depth` is set to a positive integer, only abilities reachable within that many levels from the root are included.
- `result.content` is valid source in the requested format — parseable by Mermaid or Graphviz respectively.
- `result.node_count` equals the number of distinct nodes in the graph.
- `result.edge_count` equals the number of distinct edges in the graph.

### Failure Modes

| Failure          | Condition                                                               | Effect                                       |
| ---------------- | ----------------------------------------------------------------------- | -------------------------------------------- |
| `TargetNotFound` | `target.path` does not exist.                                           | Error written to stderr; exit code non-zero. |
| `TargetIsFile`   | `target.path` is a file, not a directory.                               | Error written to stderr; exit code non-zero. |
| `ReadError`      | A file under `target` cannot be read.                                   | Error written to stderr; exit code non-zero. |
| `InvalidInclude` | An `include` value is not `scenarios`, `decisions`, or `state-machine`. | Error written to stderr; exit code non-zero. |
| `RootNotFound`   | `root` is set but does not match any ability's last path segment.       | Error written to stderr; exit code non-zero. |
| `AmbiguousRoot`  | `root` matches more than one ability's last path segment.               | Error written to stderr; exit code non-zero. |

### Notes

The graph is written to stdout. `--verbose` has no effect. `--progress` has no effect.
