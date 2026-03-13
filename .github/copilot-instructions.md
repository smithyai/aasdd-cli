This repository uses **Ability-Anchored Spec-Driven Development (AASDD)**. Read the spec before making any changes — it is the source of truth. The full methodology is at [ability-anchored-spec-driven-development](https://github.com/smithyai/ability-anchored-spec-driven-development).

## The spec

The spec structure, anatomy, and authoring rules are defined in [METHODOLOGY.md](https://github.com/smithyai/ability-anchored-spec-driven-development/blob/main/METHODOLOGY.md). Before implementing any ability, read its `ability.md` and every `concept.md` file it references. If a `decisions/` folder exists, read it before starting any implementation.

If the spec is **Draft**, the contract may shift. Do not make irreversible implementation decisions against a Draft ability unless you accept that risk. A **Stable** spec has an established contract — implementation tracks the spec version.

## Translation rules

| Spec construct | Implementation |
| --- | --- |
| Concept type | A concrete type in the language (struct, class, record, etc.) — use the exact PascalCase name from the spec |
| Ability | A module or function — the entry point uses the snake_case form of the ability name |
| Invariant | An assertion — use the language's assertion or guard mechanism for cheap checks; return an error for runtime-enforced invariants |
| Failure mode | An error variant — the PascalCase name from the spec maps directly to the variant name |

Concept names are canonical. Never rename a type or ability in the implementation — adapt casing to the language convention (`WorkspaceSnapshot` → `workspace_snapshot` in snake_case), but keep the name itself identical.

## Implementation

Follow [IMPLEMENTATION.md](https://github.com/smithyai/ability-anchored-spec-driven-development/blob/main/IMPLEMENTATION.md) for ordering rules, testing obligations, and the development cycle.

Key reminders:
- If a failure path exists in the implementation but has no corresponding spec failure mode, the spec is incomplete — add the failure mode before merging.
- If the spec includes a state machine, implement transitions exactly as defined in the Transitions table. A sub-ability may have its own `state-machine.md` scoped to its directory — treat it the same way, constrained to that ability's implementation.
- Never change implementation to match test expectations that contradict the spec.