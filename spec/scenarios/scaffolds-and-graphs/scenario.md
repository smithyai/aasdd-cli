## Scaffolds and Graphs

Scaffolding an example spec and graphing it produces a non-empty graph with the expected nodes and edges.

> `Scaffold` → `Graph`

- `Scaffold` is invoked with `example` set to `true`
- `Graph.result.node_count` is greater than zero
- `Graph.result.edge_count` is greater than zero
- `Graph.result.content` is non-empty and valid in the requested format
