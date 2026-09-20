## Graph domain

Types representing the dependency graph of a spec directory.

### GraphFormat

The output format for a rendered graph.

| Value     | Meaning                                       |
| --------- | --------------------------------------------- |
| `Mermaid` | Mermaid diagram source (Markdown-embeddable). |
| `DOT`     | Graphviz DOT language source.                 |

### NodeKind

The type of construct a graph node represents.

| Value          | Meaning                        |
| -------------- | ------------------------------ |
| `Ability`      | A top-level or nested ability. |
| `Concept`      | A concept type.                |
| `Decision`     | A decision.                    |
| `Scenario`     | A scenario.                    |
| `StateMachine` | The state machine.             |

### GraphNode

A single node in the spec dependency graph.

#### Properties

| Name    | Type                  | Description                                                                                |
| ------- | --------------------- | ------------------------------------------------------------------------------------------ |
| `id`    | text                  | Stable identifier derived from the spec path (e.g. `verify`, `verify/collect-violations`). |
| `kind`  | [NodeKind](#nodekind) | The type of construct this node represents.                                                |
| `label` | text                  | Human-readable display name (the heading from the spec file).                              |

### GraphEdge

A directed relationship between two nodes.

#### Properties

| Name       | Type | Description                                                                 |
| ---------- | ---- | --------------------------------------------------------------------------- |
| `from`     | text | The `id` of the source node.                                                |
| `to`       | text | The `id` of the target node.                                                |
| `relation` | text | The type of relationship (e.g. `sub-ability`, `input`, `output`, `traces`). |

### GraphResult

The outcome of generating a spec dependency graph.

#### Properties

| Name         | Type                                       | Description                          |
| ------------ | ------------------------------------------ | ------------------------------------ |
| `target`     | [SpecTarget](../cli/concept.md#spectarget) | The spec directory that was graphed. |
| `format`     | [GraphFormat](#graphformat)                | The output format used.              |
| `content`    | text                                       | The rendered graph source.           |
| `node_count` | number                                     | Number of nodes in the graph.        |
| `edge_count` | number                                     | Number of edges in the graph.        |
