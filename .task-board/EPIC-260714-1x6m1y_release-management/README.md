# EPIC-260714-1x6m1y: release-management

## Description
Track release-facing repository metadata for skill-ios-app-manager so tags and changelog stay aligned with the state shipped from `main`.

## Scope
Prepare changelog coverage for the current release line, verify the release cut from `main`, and publish the matching git tag and GitHub release metadata.

## Acceptance Criteria
- Current `main` state has an explicit changelog entry.
- Release tag matches the changeloged commit.
- GitHub release metadata is published for the new tag.
