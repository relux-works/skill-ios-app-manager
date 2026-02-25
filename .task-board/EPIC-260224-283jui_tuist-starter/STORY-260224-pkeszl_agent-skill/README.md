# STORY-260224-pkeszl: agent-skill

## Description
Agent-facing skill built on top of ios-app-manager CLI. Structured as instruction trees (references/) for each domain. The skill enables AI agents to fully manage Tuist-based iOS projects with Relux architecture.

Instruction trees:
1. Tuist CLI — project init, generate, build, module CRUD, entitlements, Workspace/Project.swift management
2. Relux module — scaffolding feature modules (interface+impl split), Store/Reducer/Action/State/Middleware templates, wiring
3. Dependencies via swift-ioc — DI container registration, resolution, scoping, module-level injection patterns
4. swift-httpclient — HTTP client setup, endpoint definitions, request/response patterns, middleware, auth
5. Shared Relux modules (extensible) — catalog of ready-made shared modules, how to list them, how to plug each one into a project. This tree grows as new shared modules are added

Each tree is a separate reference doc under references/. SKILL.md ties them together with triggers and routing logic.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
