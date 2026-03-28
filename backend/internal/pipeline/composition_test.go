package pipeline

import (
	"testing"

	"github.com/oarkflow/deploy/backend/internal/models"
)

func TestShouldTrigger_Success(t *testing.T) {
	link := models.PipelineLink{Condition: "success"}
	if !shouldTrigger(link, "success") {
		t.Error("should trigger on success")
	}
	if shouldTrigger(link, "failure") {
		t.Error("should not trigger on failure")
	}
}

func TestShouldTrigger_Failure(t *testing.T) {
	link := models.PipelineLink{Condition: "failure"}
	if !shouldTrigger(link, "failure") {
		t.Error("should trigger on failure")
	}
	if shouldTrigger(link, "success") {
		t.Error("should not trigger on success")
	}
}

func TestShouldTrigger_Always(t *testing.T) {
	link := models.PipelineLink{Condition: "always"}
	if !shouldTrigger(link, "success") {
		t.Error("always should trigger on success")
	}
	if !shouldTrigger(link, "failure") {
		t.Error("always should trigger on failure")
	}
	if !shouldTrigger(link, "cancelled") {
		t.Error("always should trigger on cancelled")
	}
}

func TestShouldTrigger_EmptyCondition_DefaultsToSuccess(t *testing.T) {
	link := models.PipelineLink{Condition: ""}
	if !shouldTrigger(link, "success") {
		t.Error("empty condition should default to success")
	}
	if shouldTrigger(link, "failure") {
		t.Error("empty condition should not trigger on failure")
	}
}

func TestShouldTrigger_CustomCondition(t *testing.T) {
	link := models.PipelineLink{Condition: "cancelled"}
	if !shouldTrigger(link, "cancelled") {
		t.Error("custom condition should match status exactly")
	}
	if shouldTrigger(link, "success") {
		t.Error("custom condition should not match different status")
	}
}

func TestMaxTriggerChainDepth(t *testing.T) {
	if MaxTriggerChainDepth != 10 {
		t.Errorf("MaxTriggerChainDepth = %d, want 10", MaxTriggerChainDepth)
	}
}
