# BUG-260518-utih6h: sync-team-id-to-app-and-extension-signing-settings

## Description
Repro: changing team_id then running generate project-config updates ApplicationConfiguration.developmentTeamID but leaves host let developmentTeam / DEVELOPMENT_TEAM stale and widget extension Project.swift has no DEVELOPMENT_TEAM setting

## Scope
(define bug scope / affected area)

## Acceptance Criteria
(define fix acceptance criteria)
