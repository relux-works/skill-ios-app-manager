# EPIC-260301-34qliq: module-registry

## Description
Pluggable module architecture for ios-app-manager. Central registry where each scaffolding module registers itself with metadata, dependencies, setup logic, and usage guide.

Goal: any new module = one Register() call with all metadata. CLI, two-phase setup, dependency validation, usage guides, graph visualization — all derive from the registry automatically.

Key components:
1. internal/registry/ — lightweight shared package
   - ModuleID constants (all modules reference each other through these)
   - Module struct: ID, Name, Description, Dependencies, Category, Setup, Plan, UsageGuide
   - Register/Get/All functions
   - Common utilities shared across module packages (stuff that currently duplicates)
2. Each module package (securestore, httpclient, appconfig, etc.) calls registry.Register() in init()
3. CLI commands become thin — resolve module from registry, call Plan/Setup
4. Two-phase setup: Plan() prints what will happen + usage guide → confirm → Setup() scaffolds
5. Dependency validation: before Setup(), check all Dependencies are already set up

This replaces the current scattered architecture where each module is independently wired in root.go with no shared metadata.

## Scope
(define epic scope)

## Acceptance Criteria
(define acceptance criteria)
