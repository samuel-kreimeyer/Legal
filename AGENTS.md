# legal-description

- Keep reusable parsing, normalization, geometry, and rendering logic in this package module.
- Keep interface-specific flag parsing and server startup code in app modules.
- If API payloads or XML exchange formats stabilize, extract them into `schemas/` instead of burying them in app code.
- Preserve the shared test fixtures and golden files so CLI and API behavior can be verified against the same cases.
