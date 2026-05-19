## Status
done

## Assigned To
codex

## Created
2026-05-19T13:05:52Z

## Last Update
2026-05-19T13:07:33Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Add hardcoded privileged command list and opt-in flag
- [x] Document sudo bootstrap command in README and SKILL
- [x] Run setup/help and validation

## Notes
Added scripts/setup.sh --select-xcode and --print-privileged-commands. Sudo path is opt-in only and prints/runs a hardcoded command list: sudo -v; sudo xcode-select -s /Applications/Xcode.app/Contents/Developer. No eval/user-provided shell command execution. Updated README/SKILL, re-ran normal setup to sync installed skill copy, and validated help/print commands, zsh syntax, git diff --check, task-board validate.

## Precondition Resources
(none)

## Outcome Resources
(none)
