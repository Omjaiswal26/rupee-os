package classification

import "testing"

func TestModelPresent(t *testing.T) {
	names := []string{"qwen2.5:3b", "qwen2.5:7b"}
	if !modelPresent(names, "qwen2.5:3b") {
		t.Fatal("exact")
	}
	if !modelPresent(names, "qwen2.5") {
		t.Fatal("prefix tag")
	}
	if modelPresent(names, "llama3") {
		t.Fatal("missing")
	}
}
