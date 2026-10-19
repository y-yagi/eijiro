package eijiro

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestImportAndSelect(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "dict.txt")
	content := "■apple {名} : りんご\n" +
		"■test : テスト：試験\n" +
		"■no separator line\n" +
		"■broken {brace : 壊れた\n"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	e := NewEijiro(filepath.Join(dir, "test.db"))
	if err := e.Import(src); err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	defer e.Terminate()

	tests := map[string][]string{
		"apple": {"apple {名} : りんご"},
		"test":  {"test : テスト：試験"},
	}
	for search, want := range tests {
		got, err := e.Select(search)
		if err != nil {
			t.Fatalf("Select(%q) failed: %v", search, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Select(%q) = %v, want %v", search, got, want)
		}
	}
}
