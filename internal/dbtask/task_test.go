package dbtask

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTask(t *testing.T) {
	for _, tt := range []struct{ format, content string }{
		{"json", `{"dsn":"db","driver":"postgres","sql":"select {{.TEST_DB_VALUE}}","output":"out.csv","format":"csv"}`},
		{"yaml", "dsn: db\ndriver: postgres\nsql: select {{.TEST_DB_VALUE}}\noutput: out.csv\nformat: jsonl\n"},
		{"json", `{"tasks":[{"dsn":"db","driver":"postgres","sql":"select 1","output":"out.csv"}]}`},
	} {
		t.Run(tt.format+tt.content[:1], func(t *testing.T) {
			t.Setenv("TEST_DB_VALUE", "42")
			path := filepath.Join(t.TempDir(), "task."+tt.format)
			if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
				t.Fatal(err)
			}
			tasks, err := Load(path, tt.format)
			if err != nil {
				t.Fatal(err)
			}
			if len(tasks) != 1 {
				t.Fatalf("tasks = %d", len(tasks))
			}
			if tasks[0].BatchSize != 1000 {
				t.Errorf("batch_size = %d", tasks[0].BatchSize)
			}
			if strings.Contains(tasks[0].SQL, "{{") {
				t.Errorf("SQL was not substituted: %q", tasks[0].SQL)
			}
		})
	}
}
func TestBadTask(t *testing.T) {
	for _, content := range []string{"{", `{"driver":"postgres"}`, `{"dsn":"db","driver":"postgres","sql":"select 1","output":"out","typo":3}`, `{"dsn":"db","driver":"postgres","sql":"select {{.MISSING_ENV_TEST}}","output":"out"}`} {
		path := filepath.Join(t.TempDir(), "task.json")
		os.WriteFile(path, []byte(content), 0600)
		if _, err := Load(path, "json"); err == nil {
			t.Errorf("accepted %q", content)
		}
	}
}
func TestSubstitute(t *testing.T) {
	got, err := Substitute("select '{{.X}}', {{.Y}}", func(key string) (string, bool) { return key, true })
	if err != nil || got != "select 'X', Y" {
		t.Fatalf("got %q, %v", got, err)
	}
	_, err = Substitute("select {{.MISSING}}", func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("expected missing env error")
	}
}
