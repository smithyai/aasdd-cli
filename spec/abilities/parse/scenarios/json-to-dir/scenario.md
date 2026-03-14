## JSONToDir

**Description:** Parsing a JSON snapshot into an empty target directory reconstructs the original file tree byte-for-byte.

> `Parse`

- `result.direction` is `"json-to-dir"`
- `result.file_count` equals the number of files originally serialized
- Every file from the original spec directory exists under the output directory with identical content
