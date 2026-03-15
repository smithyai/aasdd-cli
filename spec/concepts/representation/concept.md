## Representation domain

Structured types representing a parsed spec directory and the results of transfer operations.

### SpecExport

The top-level structured representation of a spec directory's semantic content. The fields of [ParsedSpecFile](#parsedspecfile) are inlined at the root level — there is no `spec` wrapper.

#### Properties

| Name            | Type                                               | Description                                                              |
| --------------- | -------------------------------------------------- | ------------------------------------------------------------------------ |
| `heading`       | text                                               | (from `ParsedSpecFile`) Top-level heading.                               |
| `aasdd`         | text                                               | (from `ParsedSpecFile`) The AASDD version.                               |
| `version`       | text                                               | (from `ParsedSpecFile`) The spec version.                                |
| `summary`       | text                                               | (from `ParsedSpecFile`) The spec summary.                                |
| `invariants`    | list of text                                       | (from `ParsedSpecFile`) Invariants from `spec.md`.                       |
| `abilities`     | list of [ParsedAbility](#parsedability)            | All top-level abilities found, each with nested sub-abilities.           |
| `concepts`      | list of [ParsedConcept](#parsedconcept)            | All concept files found.                                                 |
| `decisions`     | list of [ParsedDecision](#parseddecision)          | All decision files found.                                                |
| `scenarios`     | list of [ParsedScenario](#parsedscenario)          | All scenario files found under the scenarios/ directory.                 |
| `state_machine` | optional [ParsedStateMachine](#parsedstatemachine) | The parsed state machine, if `state-machine.md` exists at the spec root. |

### ParsedSpecFile

Structured content of a `spec.md` file.

#### Properties

| Name              | Type                                    | Description                                                             |
| ----------------- | --------------------------------------- | ----------------------------------------------------------------------- |
| `heading`         | text                                    | Top-level heading (the spec name).                                      |
| `aasdd`           | text                                    | Value of the `**AASDD:**` label.                                        |
| `version`         | text                                    | Value of the `**Version:**` label.                                      |
| `summary`         | text                                    | Summary paragraph after the version metadata.                           |
| `invariants`      | list of text                            | Bullet points under `### Invariants`, if present.                       |
| `custom_sections` | list of [CustomSection](#customsection) | Custom `###` sections after all required and optional sections, if any. |

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

| Name              | Type                                    | Description                                                                           |
| ----------------- | --------------------------------------- | ------------------------------------------------------------------------------------- |
| `heading`         | text                                    | Top-level heading (the ability name).                                                 |
| `purpose`         | text                                    | Purpose paragraph after the heading, through the first `###`.                         |
| `inputs`          | optional [Table](#table)                | Rows from the `### Inputs` table.                                                     |
| `outputs`         | optional [Table](#table)                | Rows from the `### Outputs` table.                                                    |
| `outputs_note`    | text                                    | Any prose following the `### Outputs` table, if present.                              |
| `invariants`      | list of text                            | Bullet points under `### Invariants`.                                                 |
| `failure_modes`   | optional [Table](#table)                | Rows from the `### Failure Modes` table.                                              |
| `custom_sections` | list of [CustomSection](#customsection) | Custom `###` sections after all required and optional sections, if any.               |
| `sub_abilities`   | list of [ParsedAbility](#parsedability) | Parsed sub-ability files nested under the ability directory (excluding `scenarios/`). |

### ParsedConcept

Structured content of a `concept.md` file.

#### Properties

| Name      | Type                                | Description                                                |
| --------- | ----------------------------------- | ---------------------------------------------------------- |
| `heading` | text                                | Top-level heading (the concept domain name).               |
| `intro`   | text                                | Prose between the heading and the first `###`, if present. |
| `types`   | list of [ConceptType](#concepttype) | Named types defined in the file.                           |

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

### CustomSection

An author-defined section that appears after all required and recognized optional sections in a spec file.

#### Properties

| Name      | Type | Description                                      |
| --------- | ---- | ------------------------------------------------ |
| `heading` | text | The `###` heading text.                          |
| `content` | text | Raw markdown content below the heading, trimmed. |

### ParsedScenario

Structured content of a `scenario.md` file.

#### Properties

| Name          | Type         | Description                                                     |
| ------------- | ------------ | --------------------------------------------------------------- |
| `heading`     | text         | Top-level heading (the scenario name).                          |
| `description` | text         | Description paragraph after the heading, if present.            |
| `trace`       | text         | Execution trace from the blockquote (e.g. `Scaffold → Verify`). |
| `assertions`  | list of text | Bullet points following the blockquote.                         |

### ParsedDecision

Structured content of a `decision.md` file.

#### Properties

| Name              | Type                                    | Description                                                             |
| ----------------- | --------------------------------------- | ----------------------------------------------------------------------- |
| `heading`         | text                                    | `##` heading (the decision name).                                       |
| `context`         | text                                    | Prose under `### Context`.                                              |
| `requirement`     | text                                    | Prose under `### Requirement`.                                          |
| `decision`        | text                                    | Prose under `### Decision`.                                             |
| `custom_sections` | list of [CustomSection](#customsection) | Custom `###` sections after all required and optional sections, if any. |

### ParsedStateMachine

Structured content of a `state-machine.md` file.

#### Properties

| Name                         | Type                                        | Description                                                               |
| ---------------------------- | ------------------------------------------- | ------------------------------------------------------------------------- |
| `summary`                    | text                                        | Summary paragraph after the `## State Machine` heading.                   |
| `diagram`                    | text                                        | Raw content of the mermaid code block, including fences.                  |
| `orchestrator`               | text                                        | Prose under `### Orchestrator`.                                           |
| `orchestrator_managed_state` | optional [Table](#table)                    | Table under `#### Orchestrator-Managed State`, if present.                |
| `states`                     | [Table](#table)                             | Table under `### States`.                                                 |
| `transitions`                | [Table](#table)                             | Table under `### Transitions`.                                            |
| `transition_rules`           | list of text                                | Bullet points under `### Transition Rules`, if present.                   |
| `exceptional_flows`          | list of [ExceptionalFlow](#exceptionalflow) | `#### {FlowName}` sub-sections under `### Exceptional Flows`, if present. |

### ExceptionalFlow

A named exceptional flow within a state machine's `### Exceptional Flows` section.

#### Properties

| Name      | Type | Description                                |
| --------- | ---- | ------------------------------------------ |
| `heading` | text | The `####` heading text (the flow name).   |
| `content` | text | Prose describing the exceptional behavior. |

### TransferResult

The outcome of a transfer operation (export or import).

#### Properties

| Name          | Type    | Description                                                                                           |
| ------------- | ------- | ----------------------------------------------------------------------------------------------------- |
| `file_count`  | integer | Number of spec files transferred.                                                                     |
| `output_path` | text    | Path where the output was written; empty when an export writes to the output channel instead of disk. |
