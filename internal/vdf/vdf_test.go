package vdf

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, src string) *Node {
	t.Helper()
	n, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return n
}

func TestParseLeafAndBlock(t *testing.T) {
	root := mustParse(t, `
"controller_mappings"
{
	"version"		"3"
	"title"		"d4sktop"
	"actions"
	{
		"Default"
		{
			"title"		"Desktop"
		}
	}
}`)
	cm := root.Get("controller_mappings")
	if cm == nil || !cm.IsBlock() {
		t.Fatalf("controller_mappings block missing: %+v", root)
	}
	if got := cm.Str("version"); got != "3" {
		t.Errorf("version = %q, want 3", got)
	}
	if got := cm.Get("actions").Get("Default").Str("title"); got != "Desktop" {
		t.Errorf("nested title = %q, want Desktop", got)
	}
}

func TestDuplicateKeysPreservedInOrder(t *testing.T) {
	root := mustParse(t, `"bindings" { "binding" "key_press LEFT_CONTROL, , " "binding" "key_press V, , " }`)
	got := root.Get("bindings").All("binding")
	if len(got) != 2 {
		t.Fatalf("got %d bindings, want 2", len(got))
	}
	if got[0].Value != "key_press LEFT_CONTROL, , " || got[1].Value != "key_press V, , " {
		t.Errorf("wrong order/values: %q, %q", got[0].Value, got[1].Value)
	}
}

func TestKeysAreCaseInsensitive(t *testing.T) {
	root := mustParse(t, `"Actions" { "Title" "x" }`)
	if root.Get("actions").Str("title") != "x" {
		t.Error("lookup should ignore case")
	}
}

func TestMissingKeysAreNilSafe(t *testing.T) {
	root := mustParse(t, `"a" { }`)
	if got := root.Get("nope").Get("deeper").Str("x"); got != "" {
		t.Errorf("want empty string, got %q", got)
	}
	if root.Get("nope").All("x") != nil {
		t.Error("All on nil node should be nil")
	}
	if !root.Get("a").IsBlock() || len(root.Get("a").Children) != 0 {
		t.Error("empty block should be a block with no children")
	}
}

func TestEscapesCommentsUnquotedAndConditionals(t *testing.T) {
	root := mustParse(t, `
// leading comment
"root"
{
	"quote"	"say \"hi\"\\n"  // trailing comment
	"tab"	"a\tb"
	unquoted	value
	"cond"	"win"	[$WIN32]
	"blk" [$LINUX]
	{
		"k" "v"
	}
}`)
	r := root.Get("root")
	if got := r.Str("quote"); got != `say "hi"\n` {
		t.Errorf("quote = %q", got)
	}
	if got := r.Str("tab"); got != "a\tb" {
		t.Errorf("tab = %q", got)
	}
	if got := r.Str("unquoted"); got != "value" {
		t.Errorf("unquoted = %q", got)
	}
	if got := r.Str("cond"); got != "win" {
		t.Errorf("cond = %q", got)
	}
	if got := r.Get("blk").Str("k"); got != "v" {
		t.Errorf("conditional block child = %q", got)
	}
}

func TestStringsMaySpanLinesAndContainBraces(t *testing.T) {
	root := mustParse(t, "\"a\" \"line1\nline2 { }\"")
	if got := root.Str("a"); got != "line1\nline2 { }" {
		t.Errorf("got %q", got)
	}
}

func TestErrorsReportLine(t *testing.T) {
	cases := map[string]string{
		"unterminated block":  "\"a\"\n{\n\"b\" \"c\"\n",
		"unterminated string": "\"a\" \"oops\n",
		"stray close":         "\"a\" \"b\"\n}\n",
		"key without value":   "\"a\"\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(src))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "line ") {
				t.Errorf("error should mention a line: %v", err)
			}
		})
	}
}

func TestLeadingByteOrderMarkAndCRLF(t *testing.T) {
	root := mustParse(t, "\xef\xbb\xbf\"controller_mappings\"\r\n{\r\n\t\"version\"\t\t\"3\"\r\n}\r\n")
	if got := root.Get("controller_mappings").Str("version"); got != "3" {
		t.Errorf("version = %q, want 3", got)
	}
}
