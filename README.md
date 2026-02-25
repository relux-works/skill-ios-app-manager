# tuist-starter

Minimal Go CLI starter for managing Tuist-based iOS projects.

## CI/CD (STUB)

This section is a placeholder and is not wired to a real project pipeline yet.

### CI entry points

GitHub Actions workflows call Makefile targets as CI hooks:

- `make setup`
- `make build`
- `make test`
- `make lint`

### Adding new CI steps

1. Add or update a Makefile target for the new check.
2. Call that target from the matching workflow in `.github/workflows/`.
3. Keep workflow logic thin; prefer implementation in Make targets.

### Required secrets (placeholders)

These are placeholder secret names for future iOS signing/provisioning setup:

- `TEAM_ID`
- `PROVISIONING_PROFILE_SPECIFIER`
- `PROVISIONING_PROFILE_BASE64`
