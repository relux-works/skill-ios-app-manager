package diagram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const (
	defaultDiagramName = "scaffolding-pipeline"
	externalNodeColor  = "#E8F0FE"
	featureNodeColor   = "#F5E6CC"
)

var categoryOrder = []registry.Category{
	registry.Infra,
	registry.Foundation,
	registry.Network,
	registry.Utils,
}

var categoryLabels = map[registry.Category]string{
	registry.Infra:      "Infra",
	registry.Foundation: "Foundation",
	registry.Network:    "Network",
	registry.Utils:      "Utils",
}

// GeneratePlantUML builds a PlantUML component diagram from registered modules.
func GeneratePlantUML(modules []*registry.Module) string {
	sortedModules := sortModules(modules)
	modulesByID := make(map[registry.ModuleID]*registry.Module, len(sortedModules))
	modulesByCategory := make(map[registry.Category][]*registry.Module, len(categoryOrder))

	for _, mod := range sortedModules {
		modulesByID[mod.ID] = mod
		modulesByCategory[mod.Category] = append(modulesByCategory[mod.Category], mod)
	}

	externalProducts := collectExternalProducts(sortedModules)

	var builder strings.Builder

	fmt.Fprintf(&builder, "@startuml %s\n", defaultDiagramName)
	builder.WriteString("!theme plain\n\n")
	builder.WriteString("skinparam linetype ortho\n")
	builder.WriteString("skinparam componentStyle rectangle\n\n")

	for _, product := range externalProducts {
		fmt.Fprintf(&builder, "cloud %q as %s %s\n", product, externalAlias(product), externalNodeColor)
	}
	if len(externalProducts) > 0 {
		builder.WriteString("\n")
	}

	for _, category := range categoryOrder {
		label := categoryLabels[category]
		fmt.Fprintf(&builder, "package %q {\n", label)
		for _, mod := range modulesByCategory[category] {
			fmt.Fprintf(
				&builder,
				"  component %q as %s <<module>> %s\n",
				mod.Name,
				moduleAlias(mod.ID),
				categoryColor(mod.Category),
			)
		}
		builder.WriteString("}\n\n")
	}

	builder.WriteString("package \"Feature Layer\" {\n")
	fmt.Fprintf(&builder, "  [Feature Module (relux + backend)] as FeatureModule %s\n", featureNodeColor)
	builder.WriteString("}\n\n")
	builder.WriteString("note right of FeatureModule\n")
	builder.WriteString("  Conceptual runtime module.\n")
	builder.WriteString("  Feature modules are not registered in module registry.\n")
	builder.WriteString("end note\n\n")

	for _, mod := range sortedModules {
		if mod.Category == registry.Infra {
			continue
		}
		fmt.Fprintf(&builder, "FeatureModule ..> %s\n", moduleAlias(mod.ID))
	}
	if len(sortedModules) > 0 {
		builder.WriteString("\n")
	}

	for _, mod := range sortedModules {
		for _, dep := range sortedModuleDeps(mod.Dependencies) {
			if _, ok := modulesByID[dep]; !ok {
				continue
			}
			fmt.Fprintf(&builder, "%s --> %s\n", moduleAlias(mod.ID), moduleAlias(dep))
		}
	}
	builder.WriteString("\n")

	for _, mod := range sortedModules {
		for _, product := range sortedExternalProducts(mod.ExternalDeps) {
			fmt.Fprintf(&builder, "%s --> %s\n", moduleAlias(mod.ID), externalAlias(product))
		}
	}
	builder.WriteString("\n")

	builder.WriteString("legend right\n")
	builder.WriteString("  |= Color |= Meaning |\n")
	builder.WriteString("  | <#FFF3CD> | Infra module |\n")
	builder.WriteString("  | <#D6EAF8> | Foundation module |\n")
	builder.WriteString("  | <#FFE0B2> | Network module |\n")
	builder.WriteString("  | <#E1BEE7> | Utils module |\n")
	builder.WriteString("  | <#F5E6CC> | Feature layer template |\n")
	builder.WriteString("  | <#E8F0FE> | External dependency |\n")
	builder.WriteString("  --\n")
	builder.WriteString("  Solid arrows = registry dependency\n")
	builder.WriteString("  Dashed arrows = conceptual feature runtime dependency\n")
	builder.WriteString("endlegend\n\n")

	builder.WriteString("@enduml\n")

	return builder.String()
}

func sortModules(modules []*registry.Module) []*registry.Module {
	filtered := make([]*registry.Module, 0, len(modules))
	for _, mod := range modules {
		if mod == nil {
			continue
		}
		filtered = append(filtered, mod)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Category != filtered[j].Category {
			return categoryRank(filtered[i].Category) < categoryRank(filtered[j].Category)
		}

		if filtered[i].Name != filtered[j].Name {
			return filtered[i].Name < filtered[j].Name
		}

		return filtered[i].ID < filtered[j].ID
	})

	return filtered
}

func collectExternalProducts(modules []*registry.Module) []string {
	seen := make(map[string]struct{})
	products := make([]string, 0)

	for _, mod := range modules {
		for _, dep := range mod.ExternalDeps {
			product := strings.TrimSpace(dep.Product)
			if product == "" {
				continue
			}

			if _, ok := seen[product]; ok {
				continue
			}

			seen[product] = struct{}{}
			products = append(products, product)
		}
	}

	sort.Strings(products)
	return products
}

func sortedExternalProducts(externalDeps []registry.ExternalDep) []string {
	seen := make(map[string]struct{}, len(externalDeps))
	products := make([]string, 0, len(externalDeps))

	for _, dep := range externalDeps {
		product := strings.TrimSpace(dep.Product)
		if product == "" {
			continue
		}

		if _, ok := seen[product]; ok {
			continue
		}

		seen[product] = struct{}{}
		products = append(products, product)
	}

	sort.Strings(products)
	return products
}

func sortedModuleDeps(dependencies []registry.ModuleID) []registry.ModuleID {
	sorted := append([]registry.ModuleID(nil), dependencies...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted
}

func categoryRank(category registry.Category) int {
	for index, item := range categoryOrder {
		if item == category {
			return index
		}
	}
	return len(categoryOrder)
}

func categoryColor(category registry.Category) string {
	switch category {
	case registry.Infra:
		return "#FFF3CD"
	case registry.Foundation:
		return "#D6EAF8"
	case registry.Network:
		return "#FFE0B2"
	case registry.Utils:
		return "#E1BEE7"
	default:
		return "#FAFAFA"
	}
}

func moduleAlias(moduleID registry.ModuleID) string {
	return "mod_" + sanitizeIdentifier(string(moduleID))
}

func externalAlias(product string) string {
	return "ext_" + sanitizeIdentifier(product)
}

func sanitizeIdentifier(value string) string {
	var builder strings.Builder
	for _, runeValue := range strings.ToLower(value) {
		if (runeValue >= 'a' && runeValue <= 'z') || (runeValue >= '0' && runeValue <= '9') {
			builder.WriteRune(runeValue)
			continue
		}

		builder.WriteByte('_')
	}

	sanitized := strings.Trim(builder.String(), "_")
	if sanitized == "" {
		return "node"
	}

	return sanitized
}
