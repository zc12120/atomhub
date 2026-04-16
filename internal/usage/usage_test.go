package usage

import "testing"

func TestNormalizeUsageFromParts(t *testing.T) {
	normalized := Normalize(ParsedUsage{PromptTokens: 7, CompletionTokens: 3})
	if normalized.TotalTokens != 10 {
		t.Fatalf("expected total 10, got %#v", normalized)
	}
}

func TestNormalizeUsageFromTotal(t *testing.T) {
	normalized := Normalize(ParsedUsage{PromptTokens: 4, TotalTokens: 9})
	if normalized.CompletionTokens != 5 {
		t.Fatalf("expected completion 5, got %#v", normalized)
	}
}

func TestNormalizeUsageFallback(t *testing.T) {
	normalized := Normalize(ParsedUsage{CompletionTokens: 12, TotalTokens: 12})
	if normalized.PromptTokens != 0 || normalized.CompletionTokens != 12 || normalized.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %#v", normalized)
	}
}

func TestAggregateUsageSumsNormalizedValues(t *testing.T) {
	total := Aggregate(
		ParsedUsage{PromptTokens: 1, CompletionTokens: 2},
		ParsedUsage{PromptTokens: 4, TotalTokens: 9},
	)
	if total.PromptTokens != 5 || total.CompletionTokens != 7 || total.TotalTokens != 12 {
		t.Fatalf("unexpected aggregate: %#v", total)
	}
}
