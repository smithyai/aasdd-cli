## FlatJSON

**Description:** Parsing with `flat = true` produces a top-level JSON object where every key is a forward-slash-delimited relative path and every value is a string.

> `Parse`

- `result.direction` is `"dir-to-json"`
- Every key in the output JSON object is a relative file path
- Every value in the output JSON object is a string (no nested objects)
