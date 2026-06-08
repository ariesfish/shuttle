# Domain Docs

This is a single-context repository for developer workflow skills.

## Before exploring, read these

- `CONTEXT.md` at the repo root for domain vocabulary.
- `docs/adr/` for architecture decisions that touch the area being changed.
- `docs/platform-architecture.md` for the current inference platform architecture and phase breakdown.
- `docs/prd/` for product requirements and phase-specific implementation scope.

If one of these files does not exist, proceed silently. Do not suggest creating domain docs upfront; producer skills such as `dev-align` create them lazily when terms or decisions are resolved.

## File structure

```text
/
├── CONTEXT.md
├── docs/adr/
├── docs/platform-architecture.md
├── docs/prd/
└── src/
```

## Use the glossary vocabulary

When output names a domain concept, use the term as defined in `CONTEXT.md`. Do not drift to synonyms the glossary explicitly avoids.

If a needed concept is missing from the glossary, either reconsider whether the language belongs to this project or note the gap for `dev-align`.

## Flag ADR conflicts

If output contradicts an existing ADR, surface it explicitly rather than silently overriding it.
