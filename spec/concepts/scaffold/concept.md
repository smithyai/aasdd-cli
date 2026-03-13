## Scaffold domain

Types representing the output of scaffold generation.

### ScaffoldResult

The outcome of scaffolding a new spec directory.

#### Properties

| Name | Type | Description |
| --- | --- | --- |
| `target` | [SpecTarget](../cli/concept.md#spectarget) | The directory that was created or populated. |
| `files_created` | list of text | Paths of all files written during scaffolding. |
