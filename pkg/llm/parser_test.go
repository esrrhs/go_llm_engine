package llm

import (
	"fmt"
	"strings"
	"testing"
)

func TestExtractJSON_MarkdownAndProse(t *testing.T) {
	raw := "Sure, here you go:\n```json\n{\n  \"action\": \"read_file\",\n  \"args\": {\"path\": \"a.go\"}, \n}\n```\n"
	got, err := ExtractJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"action"`) {
		t.Fatalf("unexpected json: %s", got)
	}
	act, err := ParseAction(raw)
	if err != nil {
		t.Fatal(err)
	}
	if act.Name != "read_file" {
		t.Fatalf("name=%s", act.Name)
	}
	if act.Args["path"] != "a.go" {
		t.Fatalf("args=%v", act.Args)
	}
}

func TestParseAction_Aliases(t *testing.T) {
	act, err := ParseAction(`{"tool":"done","arguments":{"summary":"ok"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if act.Name != "finish" {
		t.Fatalf("expected finish, got %s", act.Name)
	}
}

func TestExtractJSON_Unterminated(t *testing.T) {
	got, err := ExtractJSON(`{"a": 1`)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := UnmarshalFlexible(got, &m); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(m["a"]) != "1" {
		t.Fatalf("recovered %#v", m)
	}
}

func TestExtractJSON_TruncatedObjectKey(t *testing.T) {
	raw := `{
  "is_atomic": true,
  "contract": { "outputs": ["greet.go"] },
  "dod": { "description": "", "commands": ["go test ./..."], "expected_
`
	var parsed struct {
		IsAtomic bool `json:"is_atomic"`
		DoD      struct {
			Commands []string `json:"commands"`
		} `json:"dod"`
	}
	if err := UnmarshalFlexible(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if !parsed.IsAtomic || len(parsed.DoD.Commands) != 1 {
		t.Fatalf("%+v", parsed)
	}
}
