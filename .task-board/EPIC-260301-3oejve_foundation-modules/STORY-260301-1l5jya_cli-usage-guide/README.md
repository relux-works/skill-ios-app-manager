# STORY-260301-1l5jya: cli-usage-guide

## Description
Interactive setup flow: plan → confirm → scaffold.

Every setup command follows this flow:
1. Print PLAN — what files will be created, what will be patched, prerequisites
2. Print USAGE GUIDE — how to use the scaffolded code (protocols, IoC resolve, code examples)
3. Ask for confirmation — Proceed? [y/N]
4. On confirm — scaffold files, patch Registry, print done

This way AI agents see the full context BEFORE files are created. They can abort if prerequisites are missing or if the plan conflicts with current state.

Flags:
- --yes / -y — skip confirmation (for automation/pipelines)
- Default: interactive (ask for confirm)

Pattern: like terraform plan+apply in one command.

Implementation:
- Each setup package exports a Plan() function that returns plan text + usage guide
- Setup() calls Plan() first, prints it, asks confirm, then scaffolds
- Plan text includes: files to create, files to patch, prerequisites, usage examples
- Keep concise — this is stdout, not docs

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
