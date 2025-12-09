# SPEC Prefix Change Log

Date: 2025-12-09

Reason: The repository contained multiple spec folders with the same numeric prefix `001-...`, which broke `.specify` tooling that requires unique numeric prefixes per spec. To resolve automated tooling failures and follow the project's spec naming policy, the following safe renames were applied:

- `specs/001-auth-multitenancy` → `specs/002-auth-multitenancy`
- `specs/001-guest-qris-ordering` → `specs/003-guest-qris-ordering`
- `specs/001-product-inventory` → `specs/004-product-inventory`

Notes:
- No content was changed during the rename; files were moved to directories with updated numeric prefixes to avoid collisions.
- If any external references exist (CI scripts, documentation links), please update them to the new paths.

Validation:
- After the rename, run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` to confirm there are no prefix conflicts.
