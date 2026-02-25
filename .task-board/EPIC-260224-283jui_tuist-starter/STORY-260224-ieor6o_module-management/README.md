# STORY-260224-ieor6o: module-management

## Description
Full CRUD for modules via CLI. Module types: feature, kit, UI, shared, utility. IMPORTANT: product modules (feature) are split into TWO packages — interface module (protocols, public API contracts, DTOs) and implementation module (depends on interface). This enforces dependency inversion: other modules depend only on interfaces, never on implementations. Commands: tuist-starter module create (scaffolds both interface + impl for product modules, single package for utility), tuist-starter module list (shows all modules with types, interface/impl pairs, dependencies), tuist-starter module update (rename, change type, update metadata), tuist-starter module delete (removes module pair, cleans up dependencies). Each module type has its own template based on tuist-akme best patterns.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
