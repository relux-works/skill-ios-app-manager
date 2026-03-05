package dsl

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewStubQueryExecutor(t *testing.T) {
	t.Parallel()

	executor := NewStubQueryExecutor()
	result, err := executor.Execute(Expression{
		Operation: "summary",
		Params:    map[string]string{},
		Fields:    []string{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	stub, ok := result.(StubResponse)
	if !ok {
		t.Fatalf("result type = %T, want %T", result, StubResponse{})
	}

	if stub.Message != StubNotImplementedMessage {
		t.Fatalf("message = %q, want %q", stub.Message, StubNotImplementedMessage)
	}
	if stub.Kind != "query" {
		t.Fatalf("kind = %q, want %q", stub.Kind, "query")
	}
	if stub.Operation != "summary" {
		t.Fatalf("operation = %q, want %q", stub.Operation, "summary")
	}
	if !reflect.DeepEqual(stub.Params, map[string]string{}) {
		t.Fatalf("params = %#v, want empty map", stub.Params)
	}
	if !reflect.DeepEqual(stub.Fields, []string{}) {
		t.Fatalf("fields = %#v, want empty slice", stub.Fields)
	}
}

func TestNewStubMutationExecutor(t *testing.T) {
	t.Parallel()

	executor := NewStubMutationExecutor()
	result, err := executor.Execute(Expression{
		Operation: "create_module",
		Params: map[string]string{
			"name": "Auth",
			"type": "feature",
		},
		Fields: []string{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	stub, ok := result.(StubResponse)
	if !ok {
		t.Fatalf("result type = %T, want %T", result, StubResponse{})
	}
	if stub.Kind != "mutation" {
		t.Fatalf("kind = %q, want %q", stub.Kind, "mutation")
	}
	if stub.Params["name"] != "Auth" || stub.Params["type"] != "feature" {
		t.Fatalf("params = %#v, want name/type values", stub.Params)
	}
}

func TestQueryRegistryUnknownOperation(t *testing.T) {
	t.Parallel()

	executor := NewStubQueryExecutor()
	_, err := executor.Execute(Expression{Operation: "unknown", Params: map[string]string{}, Fields: []string{}})
	if err == nil {
		t.Fatal("Execute() error = nil, want unknown operation error")
	}
	if !strings.Contains(err.Error(), "unknown query operation") {
		t.Fatalf("error = %q, want unknown query operation message", err.Error())
	}
}
