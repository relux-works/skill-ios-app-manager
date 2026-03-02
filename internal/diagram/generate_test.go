package diagram

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/registry"

	_ "github.com/relux-works/ios-app-manager/internal/appconfig"
	_ "github.com/relux-works/ios-app-manager/internal/foundationplus"
	_ "github.com/relux-works/ios-app-manager/internal/httpclient"
	_ "github.com/relux-works/ios-app-manager/internal/ioc"
	_ "github.com/relux-works/ios-app-manager/internal/relux"
	_ "github.com/relux-works/ios-app-manager/internal/scaffold"
	_ "github.com/relux-works/ios-app-manager/internal/securestore"
	_ "github.com/relux-works/ios-app-manager/internal/swiftuiplus"
	_ "github.com/relux-works/ios-app-manager/internal/tokenprovider"
	_ "github.com/relux-works/ios-app-manager/internal/utilities"
)

func TestGeneratePlantUMLContainsPlantUMLMarkers(t *testing.T) {
	t.Parallel()

	output := GeneratePlantUML(registry.AllSorted())
	for _, marker := range []string{"@startuml", "@enduml"} {
		if !strings.Contains(output, marker) {
			t.Fatalf("generated diagram missing marker %q:\n%s", marker, output)
		}
	}
}

func TestGeneratePlantUMLIncludesAllRegisteredModules(t *testing.T) {
	t.Parallel()

	modules := registry.AllSorted()
	output := GeneratePlantUML(modules)

	if len(modules) == 0 {
		t.Fatal("registry.AllSorted() returned no modules")
	}

	for _, mod := range modules {
		line := fmt.Sprintf("component %q as %s", mod.Name, moduleAlias(mod.ID))
		if !strings.Contains(output, line) {
			t.Fatalf("generated diagram missing module %q line %q:\n%s", mod.Name, line, output)
		}
	}
}

func TestGeneratePlantUMLIncludesExternalCloudNodes(t *testing.T) {
	t.Parallel()

	modules := registry.AllSorted()
	output := GeneratePlantUML(modules)

	products := make(map[string]struct{})
	for _, mod := range modules {
		for _, externalDep := range mod.ExternalDeps {
			if externalDep.Product == "" {
				continue
			}
			products[externalDep.Product] = struct{}{}
		}
	}

	if len(products) == 0 {
		t.Fatal("no external dependencies found in module registry")
	}

	for product := range products {
		line := fmt.Sprintf("cloud %q as %s #E8F0FE", product, externalAlias(product))
		if !strings.Contains(output, line) {
			t.Fatalf("generated diagram missing external dependency node %q:\n%s", line, output)
		}
	}
}

func TestGeneratePlantUMLIncludesDependencyArrows(t *testing.T) {
	t.Parallel()

	modules := registry.AllSorted()
	output := GeneratePlantUML(modules)
	modulesByID := make(map[registry.ModuleID]struct{}, len(modules))
	internalArrowCount := 0
	externalArrowCount := 0

	for _, mod := range modules {
		modulesByID[mod.ID] = struct{}{}
	}

	for _, mod := range modules {
		for _, dep := range mod.Dependencies {
			if _, ok := modulesByID[dep]; !ok {
				continue
			}
			internalArrowCount++
			arrow := fmt.Sprintf("%s --> %s", moduleAlias(mod.ID), moduleAlias(dep))
			if !strings.Contains(output, arrow) {
				t.Fatalf("generated diagram missing internal dependency arrow %q:\n%s", arrow, output)
			}
		}

		for _, externalDep := range mod.ExternalDeps {
			if externalDep.Product == "" {
				continue
			}
			externalArrowCount++
			arrow := fmt.Sprintf("%s --> %s", moduleAlias(mod.ID), externalAlias(externalDep.Product))
			if !strings.Contains(output, arrow) {
				t.Fatalf("generated diagram missing external dependency arrow %q:\n%s", arrow, output)
			}
		}
	}

	if internalArrowCount == 0 {
		t.Fatal("expected at least one internal dependency arrow")
	}
	if externalArrowCount == 0 {
		t.Fatal("expected at least one external dependency arrow")
	}
}
