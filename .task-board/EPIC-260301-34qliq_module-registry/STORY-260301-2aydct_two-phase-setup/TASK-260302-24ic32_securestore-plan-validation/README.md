# TASK-260302-24ic32: securestore-plan-validation

## Description
Move access-group config validation from CLI layer into SecureStore Plan() function.

Currently internal/cli/secure_store.go has validateAccessGroup() that checks --access-group against config.AppGroups. This file will be deleted when we wire registry. The validation must move to SecureStore.Plan().

Modify internal/securestore/register.go:
- Plan() loads config from input.ProjectRoot (filepath.Join(input.ProjectRoot, "ios-app-manager.json"))
- Extracts access-group from input.ExtraArgs["access-group"]
- Validates against config.AppGroups using same logic as current validateAccessGroup:
  - If access-group empty and no config groups → error with guidance
  - If access-group empty but config has groups → error listing available groups
  - If access-group not in config groups → error saying not found + listing available
- On success, returns plan text (can remain TODO stub for now — usage-guides story will fill it in)

Error messages MUST match current test expectations:
- "--access-group is required but no app_groups defined in config"
- "--access-group is required\navailable groups in config: [group.com.example.demo]"
- "access group \"group.fake\" not found in config\navailable groups: [group.com.example.demo]"

Import config package: github.com/relux-works/ios-app-manager/internal/config
Use config.LoadConfig(filepath.Join(input.ProjectRoot, config.DefaultConfigPath))

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
