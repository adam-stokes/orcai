package pipeline_test

import (
	"testing"

	"github.com/adam-stokes/orcai/internal/pipeline"
)

func TestEvalCondition_Contains(t *testing.T) {
	if !pipeline.EvalCondition("contains:spec", "openspec output here") {
		t.Error("expected true for contains:spec")
	}
	if pipeline.EvalCondition("contains:spec", "nothing here") {
		t.Error("expected false for contains:spec")
	}
}

func TestEvalCondition_Always(t *testing.T) {
	if !pipeline.EvalCondition("always", "anything") {
		t.Error("expected always to be true")
	}
}

func TestEvalCondition_LenGt(t *testing.T) {
	if !pipeline.EvalCondition("len > 5", "hello world") {
		t.Error("expected true for len > 5 on 11-char string")
	}
	if pipeline.EvalCondition("len > 5", "hi") {
		t.Error("expected false for len > 5 on 2-char string")
	}
}

func TestEvalCondition_Matches(t *testing.T) {
	if !pipeline.EvalCondition("matches:^go", "golang is great") {
		t.Error("expected true for matches:^go")
	}
	if pipeline.EvalCondition("matches:^go", "python is great") {
		t.Error("expected false for matches:^go")
	}
}

func TestEvalCondition_Unknown(t *testing.T) {
	// Unknown expressions default to false.
	if pipeline.EvalCondition("unknown-expr", "anything") {
		t.Error("expected false for unknown expression")
	}
}
