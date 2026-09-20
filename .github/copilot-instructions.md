This repository uses **Ability-Anchored Spec-Driven Development (AASDD)**. Read the spec before making any changes — it is the source of truth. The full methodology is at [aasdd](https://github.com/smithyai/aasdd).

## The spec

The spec structure, anatomy, and authoring rules are defined in [METHODOLOGY.md](https://github.com/smithyai/aasdd/blob/main/METHODOLOGY.md). Read `spec.md` first: its Purpose, Non-Goals, and Success Criteria are the intent behind every ability. Before implementing any ability, read its `ability.md`, every `concept.md` it references, and every decision in `decisions/` whose Context names it.

If the spec version is below `1.0.0`, it has not passed the readiness gate: sections may be `_Pending._` and decisions may be `_Open._`. Do not make irreversible implementation decisions against it unless you accept that the contract may shift. At `1.0.0` and above the contract is established — implementation tracks the spec version.

## Translation rules

| Spec construct    | Implementation                                                                                                                   |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| Concept type      | A concrete type in the language (struct, class, record, etc.) — use the exact PascalCase name from the spec                      |
| Ability           | A module or function — the entry point uses the snake_case form of the ability name                                              |
| Delegated ability | A call into the delegated spec's implementation, reached as the parent spec's decision for it requires — no local implementation |
| Invariant         | An assertion — use the language's assertion or guard mechanism for cheap checks; return an error for runtime-enforced invariants |
| Failure mode      | An error variant — the PascalCase name from the spec maps directly to the variant name                                           |

Concept names are canonical. Never rename a type or ability in the implementation — adapt casing to the language convention (`WorkspaceSnapshot` → `workspace_snapshot` in snake_case), but keep the name itself identical.

## Implementation

Follow [IMPLEMENTATION.md](https://github.com/smithyai/aasdd/blob/main/IMPLEMENTATION.md) for ordering rules, testing obligations, and the development cycle, and [PROCESS.md](https://github.com/smithyai/aasdd/blob/main/PROCESS.md) for how an implementation run proceeds.

Key reminders:
- After any change to a spec file, run `aasdd verify <spec-dir>` to confirm the spec conforms to the methodology and conventions. Fix all reported violations before proceeding.
- Derive tests from the spec before writing implementation: one per invariant, one per failure mode, one per scenario, and one for each Idempotency section.
- Implement sub-abilities in the order their parent's Composition table defines. If the spec includes a state machine, implement transitions exactly as defined in the Transitions table.
- If a failure path exists in the implementation but has no corresponding spec failure mode, the spec is incomplete — add the failure mode before merging.
- If the spec does not answer a question the implementation needs answered, do not guess: add an open decision under `decisions/` whose Context names the ability, skip that ability and everything that depends on it, and continue with the rest.
- If new behavior, inputs, outputs, or options are added that are not reflected in the spec (abilities, concepts, or decisions), update the spec first — implementation follows the spec, not the other way around.
- Never change implementation to match test expectations that contradict the spec.

## This repository

The spec lives in [`spec/`](../spec/) and is self-hosted: `make verify` checks it with the local build, and `tests/self` exports, imports, and verifies it on every test run. The spec files are kept in the canonical padded form so that the round-trip is byte-identical; run `aasdd export` and `aasdd import` to regenerate them if a table drifts. Every rule in `load_rule_set` corresponds to a rule or readiness condition in the methodology documents; when the methodology changes, the rule set changes with it, and the examples in the methodology repository are the reference specs the rule set must accept.
