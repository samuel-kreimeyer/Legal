# Issue: Could Not Detect Project Language

**Type:** type
**Severity:** warning
**Tool:** check-types
**Detected:** 2026-01-10T16:22:45.191947Z

## Summary
Unable to determine project language for type checking.

## Evidence
No recognized configuration files found (pyproject.toml, Cargo.toml, go.mod, tsconfig.json, package.json).
No source files with recognized extensions found in src/.

## Impact
Static analysis cannot run without knowing the project language.

## Recommended Action
Ensure the project has appropriate configuration files and source code in src/ directory.

## Automation
- Detectable: yes
- Auto-fixable: no