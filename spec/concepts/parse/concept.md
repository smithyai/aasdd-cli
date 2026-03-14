## Parse domain

Types used by the `Parse` ability.

### SpecSnapshot

A portable, path-indexed representation of a spec directory's file contents.

#### Properties

| Name    | Type               | Description                                                                                              |
| ------- | ------------------ | -------------------------------------------------------------------------------------------------------- |
| `files` | map of text → text | Maps each file's path (relative to the spec root, forward-slash delimited) to its full text content. |

A `SpecSnapshot` is serialized to JSON in one of two forms controlled by the `flat` input:

- **Flat** (`flat = true`): the `files` map is written directly as a JSON object — `{"path/to/file": "content", ...}`.
- **Nested** (`flat = false`, default): files are grouped into a recursive directory-tree JSON object where intermediate keys are directory names and leaf string values are file contents.

Both forms are semantically equivalent and interchangeable — `Parse` accepts either form for JSON-to-dir without requiring the `flat` flag.

> **Note:** `SpecSnapshot` preserves only text file contents. Binary files are not supported.

### ParseResult

The outcome of a `Parse` operation.

#### Properties

| Name          | Type                              | Description                                                       |
| ------------- | --------------------------------- | ----------------------------------------------------------------- |
| `direction`   | [ParseDirection](#parsedirection) | Whether this was a dir-to-JSON or JSON-to-dir operation.          |
| `file_count`  | integer                           | Number of files in the snapshot.                                  |
| `output_path` | text                              | Path where output was written; empty string if written to stdout. |

### ParseDirection

| Value           | Meaning                                                     |
| --------------- | ----------------------------------------------------------- |
| `"dir-to-json"` | Source was a spec directory; output is a JSON snapshot.     |
| `"json-to-dir"` | Source was a JSON snapshot; output is a spec directory.     |
