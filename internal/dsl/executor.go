package dsl

import (
	"fmt"
	"sort"
	"strings"
)

const StubNotImplementedMessage = "not implemented"

// QueryExecutor executes read operations for parsed expressions.
type QueryExecutor interface {
	Execute(Expression) (any, error)
}

// MutationExecutor executes write operations for parsed expressions.
type MutationExecutor interface {
	Execute(Expression) (any, error)
}

// QueryHandler is the function signature for a query operation handler.
type QueryHandler func(Expression) (any, error)

// MutationHandler is the function signature for a mutation operation handler.
type MutationHandler func(Expression) (any, error)

// StubResponse is returned by scaffold handlers until real implementations are added.
type StubResponse struct {
	Message   string            `json:"message"`
	Kind      string            `json:"kind"`
	Operation string            `json:"operation"`
	Params    map[string]string `json:"params"`
	Fields    []string          `json:"fields"`
}

// QueryHandlerRegistry stores query handlers by operation name.
type QueryHandlerRegistry struct {
	handlers map[string]QueryHandler
}

// MutationHandlerRegistry stores mutation handlers by operation name.
type MutationHandlerRegistry struct {
	handlers map[string]MutationHandler
}

var stubQueryOperations = []string{
	"summary",
	"modules",
	"get",
	"deps",
	"config",
	"entitlements",
}

var stubMutationOperations = []string{
	"create_module",
	"delete_module",
	"add_dep",
	"remove_dep",
	"add_entitlement",
	"init",
}

// NewQueryHandlerRegistry creates an empty query handler registry.
func NewQueryHandlerRegistry() *QueryHandlerRegistry {
	return &QueryHandlerRegistry{handlers: map[string]QueryHandler{}}
}

// NewMutationHandlerRegistry creates an empty mutation handler registry.
func NewMutationHandlerRegistry() *MutationHandlerRegistry {
	return &MutationHandlerRegistry{handlers: map[string]MutationHandler{}}
}

// Register registers a query handler for an operation.
func (r *QueryHandlerRegistry) Register(operation string, handler QueryHandler) {
	if r == nil || handler == nil {
		return
	}

	op := strings.TrimSpace(operation)
	if op == "" {
		return
	}

	r.handlers[op] = handler
}

// Register registers a mutation handler for an operation.
func (r *MutationHandlerRegistry) Register(operation string, handler MutationHandler) {
	if r == nil || handler == nil {
		return
	}

	op := strings.TrimSpace(operation)
	if op == "" {
		return
	}

	r.handlers[op] = handler
}

// Execute executes a query expression with the matching registered handler.
func (r *QueryHandlerRegistry) Execute(expression Expression) (any, error) {
	if r == nil {
		return nil, fmt.Errorf("query handler registry is nil")
	}

	op := strings.TrimSpace(expression.Operation)
	if op == "" {
		return nil, fmt.Errorf("operation is required")
	}

	handler, ok := r.handlers[op]
	if !ok {
		return nil, fmt.Errorf("unknown query operation %q (registered: %s)", op, strings.Join(r.operations(), ", "))
	}

	return handler(expression)
}

// Execute executes a mutation expression with the matching registered handler.
func (r *MutationHandlerRegistry) Execute(expression Expression) (any, error) {
	if r == nil {
		return nil, fmt.Errorf("mutation handler registry is nil")
	}

	op := strings.TrimSpace(expression.Operation)
	if op == "" {
		return nil, fmt.Errorf("operation is required")
	}

	handler, ok := r.handlers[op]
	if !ok {
		return nil, fmt.Errorf("unknown mutation operation %q (registered: %s)", op, strings.Join(r.operations(), ", "))
	}

	return handler(expression)
}

// NewStubQueryExecutor creates query handlers for all scaffold operations.
func NewStubQueryExecutor() QueryExecutor {
	registry := NewQueryHandlerRegistry()
	for _, operation := range stubQueryOperations {
		registry.Register(operation, makeStubQueryHandler())
	}
	return registry
}

// NewStubMutationExecutor creates mutation handlers for all scaffold operations.
func NewStubMutationExecutor() MutationExecutor {
	registry := NewMutationHandlerRegistry()
	for _, operation := range stubMutationOperations {
		registry.Register(operation, makeStubMutationHandler())
	}
	return registry
}

func makeStubQueryHandler() QueryHandler {
	return func(expression Expression) (any, error) {
		return StubResponse{
			Message:   StubNotImplementedMessage,
			Kind:      "query",
			Operation: expression.Operation,
			Params:    cloneParams(expression.Params),
			Fields:    cloneFields(expression.Fields),
		}, nil
	}
}

func makeStubMutationHandler() MutationHandler {
	return func(expression Expression) (any, error) {
		return StubResponse{
			Message:   StubNotImplementedMessage,
			Kind:      "mutation",
			Operation: expression.Operation,
			Params:    cloneParams(expression.Params),
			Fields:    cloneFields(expression.Fields),
		}, nil
	}
}

func cloneParams(params map[string]string) map[string]string {
	cloned := map[string]string{}
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}

func cloneFields(fields []string) []string {
	if len(fields) == 0 {
		return []string{}
	}
	return append([]string(nil), fields...)
}

func (r *QueryHandlerRegistry) operations() []string {
	ops := make([]string, 0, len(r.handlers))
	for op := range r.handlers {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return ops
}

func (r *MutationHandlerRegistry) operations() []string {
	ops := make([]string, 0, len(r.handlers))
	for op := range r.handlers {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return ops
}
