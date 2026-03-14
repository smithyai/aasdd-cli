## Export domain

Types used by the `Export` and `Import` abilities.

### SpecExport

The top-level structured representation of a spec directory's semantic content. The fields of [ParsedSpecFile](#parsedspecfile) are inlined at the root level — there is no `spec` wrapper.

#### Properties

| Name         | Type                                      | Description                                                    |
| ------------ | ----------------------------------------- | -------------------------------------------------------------- |
| `heading`    | text                                      | (from `ParsedSpecFile`) Top-level heading.                     |
| `aasdd`      | text                                      | (from `ParsedSpecFile`) The AASDD version.                     |
| `version`    | text                                      | (from `ParsedSpecFile`) The spec version.                      |
| `summary`    | text                                      | (from `ParsedSpecFile`) The spec summary.                      |
| `invariants` | list of text                              | (from `ParsedSpecFile`) Invariants from `spec.md`.             |
| `abilities`  | list of [ParsedAbility](#parsedability)   | All top-level abilities found, each with nested sub-abilities. |
| `concepts`   | list of [ParsedConcept](#parsedconcept)   | All concept files found.                                       |
| `decisions`  | list of [ParsedDecision](#parseddecision) | All decision files found.                                      |
| `scenarios`  | list of [ParsedScenario](#parsedscenario) | All scenario files found under the scenarios/ directory.       |

### ParsedSpecFile

Structured content of a `spec.md` file.

#### Properties

| Name         | Type         | Description                                       |
| ------------ | ------------ | ------------------------------------------------- |
| `heading`    | text         | Top-level heading (the spec name).                |
| `aasdd`      | text         | Value of the `**AASDD:**` label.                  |
| `version`    | text         | Value of the `**Version:**` label.                |
| `summary`    | text         | Value of the `**Summary:**` label.                |
| `invariants` | list of text | Bullet points under `### Invariants`, if present. |

### Table

A parsed Markdown table with its original column order preserved.

#### Properties

| Name      | Type                    | Description                                     |
| --------- | ----------------------- | ----------------------------------------------- |
| `headers` | list of text            | Column names in their original order.           |
| `rows`    | list of map text → text | Each row as a map of column name to cell value. |

### ParsedAbility

Structured content of an `ability.md` file.

#### Properties

| Name            | Type                                    | Description                                                                           |
| --------------- | --------------------------------------- | ------------------------------------------------------------------------------------- |
| `path`          | text                                    | File path relative to the spec root, forward-slash delimited.                         |
| `heading`       | text                                    | Top-level heading (the ability name).                                                 |
| `purpose`       | text                                    | Full content of the `**Purpose:**` label through the first `###`, newlines preserved. |
| `inputs`        | optional [Table](#table)                | Rows from the `### Inputs` table.                                                     |
| `outputs`       | optional [Table](#table)                | Rows from the `### Outputs` table.                                                    |
| `outputs_note`  | text                                    | Any prose following the `### Outputs` table, if present.                              |
| `invariants`    | list of text                            | Bullet points under `### Invariants`.                                                 |
| `failure_modes` | optional [Table](#table)                | Rows from the `### Failure Modes` table.                                              |
| `notes`         | text                                    | Prose under `### Notes`, if present.                                                  |
| `visualization` | text                                    | Mermaid diagram source under `### Visualization`, if present.                         |
| `sub_abilities` | list of [ParsedAbility](#parsedability) | Parsed sub-ability files nested under the ability directory (excluding `scenarios/`). |

### ParsedConcept

Structured content of a `concept.md` file.

#### Properties

| Name      | Type                                | Description                                                   |
| --------- | ----------------------------------- | ------------------------------------------------------------- |
| `path`    | text                                | File path relative to the spec root, forward-slash delimited. |
| `heading` | text                                | Top-level heading (the concept domain name).                  |
| `intro`   | text                                | Prose between the heading and the first `###`, if present.    |
| `types`   | list of [ConceptType](#concepttype) | Named types defined in the file.                              |

### ConceptType

A named type within a `concept.md` file.

#### Properties

| Name                 | Type                     | Description                                                               |
| -------------------- | ------------------------ | ------------------------------------------------------------------------- |
| `name`               | text                     | The type name (`###` heading).                                            |
| `description`        | text                     | Prose between the heading and the table or `#### Properties`, if present. |
| `properties_heading` | boolean                  | `true` when the table is preceded by a `#### Properties` sub-heading.     |
| `note`               | text                     | Content of a `> **Note:**` block after the table, if present.             |
| `properties`         | optional [Table](#table) | Table rows (property definitions or enum values).                         |

### ParsedScenario

Structured content of a `scenario.md` file.

#### Properties

| Name          | Type         | Description                                                     |
| ------------- | ------------ | --------------------------------------------------------------- |
| `path`        | text         | File path relative to the spec root, forward-slash delimited.   |
| `heading`     | text         | Top-level heading (the scenario name).                          |
| `description` | text         | Value of `**Description:**`, if present.                        |
| `trace`       | text         | Execution trace from the blockquote (e.g. `Scaffold → Verify`). |
| `assertions`  | list of text | Bullet points following the blockquote.                         |

### ParsedDecision

Structured content of a `decision.md` file.

#### Properties

| Name          | Type | Description                                                   |
| ------------- | ---- | ------------------------------------------------------------- |
| `path`        | text | File path relative to the spec root, forward-slash delimited. |
| `heading`     | text | `##` heading (the decision name).                             |
| `context`     | text | Prose under `### Context`.                                    |
| `requirement` | text | Prose under `### Requirement`.                                |
| `decision`    | text | Prose under `### Decision`.                                   |

### ExportResult

The outcome of an `Export` operation.

#### Properties

| Name          | Type    | Description                                                                  |
| ------------- | ------- | ---------------------------------------------------------------------------- |
| `file_count`  | integer | Number of spec files included in the export.                                 |
| `output_path` | text    | Path where the export was written; empty when written to the output channel. |

### ImportResult

The outcome of an `Import` operation.

#### Properties

| Name          | Type    | Description                                        |
| ------------- | ------- | -------------------------------------------------- |
| `file_count`  | integer | Number of files written to the output directory.   |
| `output_path` | text    | Path of the directory the files were written into. |
