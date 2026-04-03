# legal-description

Shared Go library for parsing parcel geometry from DXF, IFC, and LandXML inputs and rendering legal-description text.

## Layout

- `pkg/`: reusable geometry, parsing, normalization, rendering, and API support packages.
- `tests/`: shared fixtures and golden outputs used by package and app verification.
- `docs/`: preserved source development and reference notes.
- `scripts/`: preserved source smoke-test helpers.
- `Test/`: preserved legacy tests that are excluded from default builds behind the `legacy` build tag.
- `README.source.md` and `go.source.mod`: preserved source metadata before monorepo path normalization.

## Notes

- This package is the canonical home for reusable legal-description logic in the monorepo.
- User-facing entrypoints live in [`/home/sam/Projects/Curatores Viarum/apps/legal-description-cli`](/home/sam/Projects/Curatores%20Viarum/apps/legal-description-cli) and [`/home/sam/Projects/Curatores Viarum/apps/legal-description-api`](/home/sam/Projects/Curatores%20Viarum/apps/legal-description-api).
- The legacy `legalxml` entrypoint was only documented in the source repo, so it remains a placeholder app until an actual command is implemented.
