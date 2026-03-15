## GraphsInDotFormat

Scaffolding an example spec and graphing it with DOT format produces valid Graphviz DOT source with the expected nodes and edges.

> `Scaffold` → `Graph`

- `Scaffold` is invoked with `example` set to `true`
- `Graph` is invoked with `format` set to `DOT`
- `Graph.result.content` starts with `digraph`
- `Graph.result.node_count` is greater than zero
- `Graph.result.edge_count` is greater than zero
