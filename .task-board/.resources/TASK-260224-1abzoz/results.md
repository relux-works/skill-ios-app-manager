Implemented IoC wiring templates and tests.

Files:
- internal/relux/templates/ioc_registration.swift.tmpl
- internal/relux/templates/ioc_resolver.swift.tmpl
- internal/relux/templates/composition_root.swift.tmpl
- internal/relux/template_engine.go
- internal/relux/template_engine_test.go
- internal/relux/templates_test.go
- internal/relux/testdata/golden/ioc_registration.swift
- internal/relux/testdata/golden/ioc_resolver.swift
- internal/relux/testdata/golden/composition_root.swift
- testdata/relux/ioc_registration.golden
- testdata/relux/ioc_resolver.golden
- testdata/relux/composition_root.golden

Verification:
- GOCACHE=$(pwd)/.tmp/gocache GOMODCACHE=$(pwd)/.tmp/gomodcache go test ./internal/relux/...
- GOCACHE=$(pwd)/.tmp/gocache GOMODCACHE=$(pwd)/.tmp/gomodcache go test ./...
- GOCACHE=$(pwd)/.tmp/gocache GOMODCACHE=$(pwd)/.tmp/gomodcache go vet ./...