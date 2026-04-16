package anthropic

import "testing"

func TestParseModelsResponseParsesModels(t *testing.T) {
	body := []byte(`{"data":[{"id":"claude-3-5-sonnet-latest"},{"id":"claude-3-7-sonnet-latest"}]}`)
	models, err := ParseModels(body)
	if err != nil {
		t.Fatalf("parse models: %v", err)
	}
	if len(models) != 2 || models[0] != "claude-3-5-sonnet-latest" || models[1] != "claude-3-7-sonnet-latest" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestParseModelsResponseSkipsEmptyIDs(t *testing.T) {
	body := []byte(`{"data":[{"id":"claude-3-5-sonnet-latest"},{"id":""},{"id":"claude-3-5-sonnet-latest"}]}`)
	models, err := ParseModels(body)
	if err != nil {
		t.Fatalf("parse models: %v", err)
	}
	if len(models) != 1 || models[0] != "claude-3-5-sonnet-latest" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestParseModelsResponseRejectsInvalidJSON(t *testing.T) {
	if _, err := ParseModels([]byte(`{"data":`)); err == nil {
		t.Fatalf("expected parse error")
	}
}
