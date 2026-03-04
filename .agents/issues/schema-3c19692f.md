# Issue: No Configuration Files Detected

**Type:** schema
**Severity:** info
**Tool:** check-schema
**Detected:** 2026-01-10T16:22:45.191299Z

## Summary
No common configuration files found in project root.

## Evidence
Checked for: pyproject.toml, Cargo.toml, go.mod, package.json, and other common config files.
None found.

## Impact
Configuration files help identify the project type and provide metadata. While not strictly required, most projects should have at least one configuration file.

## Recommended Action
Consider adding a configuration file appropriate for your project type:

- Python: `pyproject.toml`
- Rust: `Cargo.toml`
- Go: `go.mod`
- JavaScript/TypeScript: `package.json`

## Automation
- Detectable: yes
- Auto-fixable: no