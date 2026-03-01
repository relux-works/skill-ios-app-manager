# STORY-260301-2aydct: two-phase-setup

## Description
Implement two-phase setup flow using registry metadata.

Flow:
1. CLI resolves module from registry by ID
2. Calls module.Plan(input) — returns text: files to create, files to patch, prerequisites, usage guide
3. Checks Dependencies — all must be already set up (check Registry.swift or marker files)
4. Prints plan + usage guide to stdout
5. Asks Proceed? [y/N]
6. On confirm — calls module.Setup(input)
7. Prints done

Flags:
- --yes / -y — skip confirmation
- --dry-run — print plan only, dont scaffold

Generic implementation in CLI layer — works for any registered module automatically. No per-module CLI code needed beyond what registry provides.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
