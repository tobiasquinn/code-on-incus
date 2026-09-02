package session

import (
	"encoding/json"
	"testing"
)

// TestStripJSONC_CommentsAndTrailingCommas covers the opencode.jsonc merge path:
// comments and trailing commas must be stripped before the plain-JSON parse so
// host config survives the sandbox-settings merge.
func TestStripJSONC_CommentsAndTrailingCommas(t *testing.T) {
	in := `{
  // line comment
  "theme": "dark", /* block */ "url": "http://x//y", // trailing
  "list": [ "a", "b", ],
  "nested" : { "still": "valid", },
}`
	settings := map[string]interface{}{"permission": map[string]interface{}{"*": "allow"}}

	res, parseErr, err := mergeJSONSettings([]byte(in), settings)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if parseErr != nil {
		t.Errorf("parseErr = %v (JSONC should normalize)", parseErr)
	}

	var parsed map[string]interface{}
	if jerr := json.Unmarshal(res, &parsed); jerr != nil {
		t.Fatalf("result not plain JSON: %v", jerr)
	}
	if parsed["theme"] != "dark" || parsed["url"] != "http://x//y" || parsed["list"] == nil {
		t.Errorf("user content lost: %v", parsed)
	}
	perm, ok := parsed["permission"].(map[string]interface{})
	if !ok || perm["*"] != "allow" {
		t.Errorf("sandbox settings missing: %v", parsed["permission"])
	}
}

// TestStripJSONC_StillInvalid reports non-JSONC garbage as a parse error
// (callers warn and overwrite) instead of silently dropping the file.
func TestStripJSONC_StillInvalid(t *testing.T) {
	in := `{ "weird": this is not jsonc }`
	_, parseErr, err := mergeJSONSettings([]byte(in), map[string]interface{}{"x": 1})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if parseErr == nil {
		t.Error("parseErr = nil for actually-invalid content")
	}
}

// TestStripJSONC_BlockCommentMultiLine removed together across lines.
func TestStripJSONC_BlockCommentMultiLine(t *testing.T) {
	in := `{
  /* multi
     line */
  "ok": 1
}`
	settings := map[string]interface{}{}
	res, parseErr, err := mergeJSONSettings([]byte(in), settings)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if parseErr != nil {
		t.Fatalf("parseErr = %v", parseErr)
	}
	var parsed map[string]interface{}
	if jerr := json.Unmarshal(res, &parsed); jerr != nil {
		t.Fatalf("unmarshal: %v", jerr)
	}
	if parsed["ok"] != float64(1) {
		t.Errorf("parsed = %v", parsed)
	}
}

// TestStripJSONC_TrailingCommaAndEmptyString ensures empty strings and quoted
// separators survive the trailing-comma scan.
func TestStripJSONC_TrailingCommaAndEmptyString(t *testing.T) {
	in := `{ "a": "", "b": "}", "c": "]", "d": 1, }`
	settings := map[string]interface{}{}
	res, parseErr, err := mergeJSONSettings([]byte(in), settings)
	if err != nil || parseErr != nil {
		t.Fatalf("err %v parseErr %v", err, parseErr)
	}
	var parsed map[string]interface{}
	if jerr := json.Unmarshal(res, &parsed); jerr != nil {
		t.Fatalf("unmarshal: %v", jerr)
	}
	if parsed["a"] != "" || parsed["b"] != "}" || parsed["c"] != "]" || parsed["d"] != float64(1) {
		t.Errorf("parsed = %v", parsed)
	}
}

// TestStripJSONC_EscapedQuoteInString handles escaped quotes adjacent to commas.
func TestStripJSONC_EscapedQuoteInString(t *testing.T) {
	in := `{"a": "say \"hi\",", "b": 1}`
	settings := map[string]interface{}{}
	res, parseErr, err := mergeJSONSettings([]byte(in), settings)
	if err != nil || parseErr != nil {
		t.Fatalf("err %v parseErr %v", err, parseErr)
	}
	var parsed map[string]interface{}
	if jerr := json.Unmarshal(res, &parsed); jerr != nil {
		t.Fatalf("unmarshal: %v", jerr)
	}
	if parsed["a"] != `say "hi",` || parsed["b"] != float64(1) {
		t.Errorf("parsed = %v", parsed)
	}
}
