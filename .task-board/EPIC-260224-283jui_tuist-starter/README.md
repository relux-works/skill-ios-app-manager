# Tuist CLI Tool

## Description
CLI tool (Go) for managing Tuist-based iOS projects. Creates new projects, adds modules, manages entitlements, configures dependencies — all through a command-line interface. Built with agent-facing API pattern (DSL queries + mutations) and go-testing-tools for tests. Best patterns stolen from tuist-akme, minus the garbage. Future: agent-facing skill built on top of this CLI.

## Scope
1. CLI tool (Swift) that can: setup new Tuist project from scratch, add modules (feature/kit/UI/etc), configure inter-module dependencies, manage external dependencies. 2. Clean project structure based on tuist-akme patterns. 3. Foundation for future agent skill.

## Acceptance Criteria
- CLI tool builds and runs on macOS
- Can create a new Tuist project with proper structure
- Can add typed modules (feature, kit, UI, etc)
- Can configure module dependencies
- Generated projects build successfully with tuist generate
- Patterns match best practices from tuist-akme analysis
