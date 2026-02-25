# Product Forensics: tuist-akme

## Target
/Users/alexis/src/relux-works/tuist-akme/

## Goal
Thoroughly analyze the tuist-akme project to understand its Tuist-based iOS project structure, patterns, and conventions. Findings will be used to bootstrap a new project (tuist-starter) following the same patterns.

## Methodology (4-layer MapReduce)

### L1 Recon — scan the repo structure
- Directory tree, file counts by type
- Key config files (Tuist.swift, Workspace.swift, Package.swift, Project.swift files)
- Module organization (Apps/, Modules/, etc.)
- Build system setup (Makefile, Scripts/)

### L2 Deep-Dive — read and analyze key files
- Tuist configuration (Tuist.swift, Workspace.swift, any Project.swift files in modules)
- Module structure — how modules are organized, what each module contains
- Dependency graph — how modules depend on each other
- App targets — what apps exist, how they're configured
- Package.swift — external dependencies
- Build scripts and automation (Makefile, Scripts/)
- Tuist plugins (TuistPlugins/)
- Code patterns — Swift style, architecture patterns used

### L3 Domain Synthesis — group findings by domain
- Project Configuration domain (Tuist setup, workspace, project generation)
- Module Architecture domain (module types, boundaries, dependencies)
- Build & CI domain (scripts, automation, Makefile)
- External Dependencies domain (SPM packages, plugins)

### L4 Product Synthesis — final report
- Executive summary
- Architecture overview
- Key patterns and conventions
- Module catalog with purposes
- Dependency map
- Recommendations for replication in a new starter project

## Output
Write the FULL report to: /Users/alexis/src/relux-works/tuist-starter/.research/260224_tuist-akme-forensics.md
Create the .research/ directory if it doesn't exist.

The report must be comprehensive, in English, well-structured with markdown headers, and include code snippets from key configuration files where relevant.

IMPORTANT: This is a RESEARCH task. Read files extensively, do NOT modify any files in tuist-akme. Only write the report file in tuist-starter.
