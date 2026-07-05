package wire

import (
	"strings"
	"testing"
)

func TestUnifiedDiffReadable(t *testing.T) {
	before := []string{
		"package main",
		"",
		"import (",
		`	"log"`,
		`	"net/http"`,
		")",
		"",
		"func main() {",
		`	log.Println("listening")`,
		`	http.ListenAndServe(":8080", nil)`,
		"}",
	}
	after := []string{
		"package main",
		"",
		"import (",
		`	"log"`,
		`	"net/http"`,
		"",
		`	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"`,
		")",
		"",
		"func main() {",
		`	if err := debugviz.ConfigureFromEnv(); err != nil {`,
		`		log.Fatalf("debugviz: %v", err)`,
		"	}",
		`	log.Println("listening")`,
		`	http.ListenAndServe(":8080", nil)`,
		"}",
	}

	got := unifiedDiff("demo/http/main.go", before, after)
	if got == "" {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(got, "--- a/demo/http/main.go") {
		t.Fatalf("missing file header:\n%s", got)
	}
	if !strings.Contains(got, "@@") {
		t.Fatalf("missing hunk header:\n%s", got)
	}
	if strings.Contains(got, "- )\n+\t\"github.com") {
		t.Fatalf("diff lines are misaligned:\n%s", got)
	}
	if !strings.Contains(got, "+	if err := debugviz.ConfigureFromEnv()") {
		t.Fatalf("missing configure injection:\n%s", got)
	}
}

func TestUnifiedDiffNoChanges(t *testing.T) {
	lines := []string{"package main", "", "func main() {}"}
	if diff := unifiedDiff("main.go", lines, lines); diff != "" {
		t.Fatalf("expected empty diff, got:\n%s", diff)
	}
}
