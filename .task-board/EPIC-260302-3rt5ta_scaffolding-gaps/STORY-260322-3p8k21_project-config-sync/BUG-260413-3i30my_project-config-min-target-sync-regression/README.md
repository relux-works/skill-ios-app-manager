# BUG-260413-3i30my: project-config-min-target-sync-regression

## Description
Repro in x-platform-airdrop: after changing ios-app-manager.json min_target from 26.0 to 18.0, ios-app-manager generate project-config reports min-target already up to date and leaves root Project.swift at 26.0. The same run also rewrites root Package.swift and drops the existing #if TUIST PackageSettings strictness block. Manual patch of Project.swift and package manifests was required.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
(define fix acceptance criteria)
