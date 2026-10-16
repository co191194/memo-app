package storage_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/co191194/memo-app/internal/memo"
	"github.com/co191194/memo-app/internal/storage"
)

type Memo = memo.Memo
type StorageOperatorImpl = storage.StorageOperatorImpl

func TestSaveAndLoadMemos(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "memos.json")

	now := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	expected := []Memo{
		{
			ID:        1,
			Title:     "I study Go lang",
			Body:      "using testing package",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	mo := StorageOperatorImpl{}

	if err := mo.SaveMemos(filePath, expected); err != nil {
		t.Fatalf("SaveMemos() error = %v", err)
	}

	actual, err := mo.LoadMemos(filePath)
	if err != nil {
		t.Fatalf("LoadMemos() error = %v", err)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("LoadMemos() = %v, expected = %v", actual, expected)
	}
}

func TestLoadMemos_FileDoesNotExist(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-exist.json")

	mo := StorageOperatorImpl{}

	actual, err := mo.LoadMemos(filePath)
	if err != nil {
		t.Fatalf("LoadMemos() error = %v", err)
	}

	if len(actual) != 0 {
		t.Errorf("len(LoadMemos()) = %d, expected = 0", len(actual))
	}
}

func TestLoadMemos_InvalidJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{"syntax error", `[{`},
		{"null document", `null`},
		{"non-array document", `{}`},
		{"second value", `[] {}`},
		{"trailing garbage", `[] not-json`},
		{"null element", `[null]`},
		{"non-object element", `[1]`},
		{"missing id", `[{"title":"Title","body":"","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"missing title", `[{"id":1,"body":"","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"missing body", `[{"id":1,"title":"Title","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"missing created at", `[{"id":1,"title":"Title","body":"","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"missing updated at", `[{"id":1,"title":"Title","body":"","created_at":"2026-01-01T00:00:00Z"}]`},
		{"null field", `[{"id":null,"title":"Title","body":"","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"null title", `[{"id":1,"title":null,"body":"","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"null body", `[{"id":1,"title":"Title","body":null,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"null date", `[{"id":1,"title":"Title","body":"","created_at":null,"updated_at":"2026-01-01T00:00:00Z"}]`},
		{"wrong type", `[{"id":"1","title":"Title","body":"","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"wrong title type", `[{"id":1,"title":1,"body":"","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"wrong body type", `[{"id":1,"title":"Title","body":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"invalid date", `[{"id":1,"title":"Title","body":"","created_at":"yesterday","updated_at":"2026-01-01T00:00:00Z"}]`},
		{"invalid updated date", `[{"id":1,"title":"Title","body":"","created_at":"2026-01-01T00:00:00Z","updated_at":"yesterday"}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memos.json")
			before := []byte(tc.data)
			if err := os.WriteFile(path, before, 0600); err != nil {
				t.Fatal(err)
			}

			got, err := (&StorageOperatorImpl{}).LoadMemos(path)
			if err == nil || got != nil {
				t.Errorf("LoadMemos() = %v, %v; want nil and an error", got, err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("invalid JSON was modified: before = %q, after = %q", before, after)
			}
		})
	}
}

func TestLoadMemos_EmptyArrayWithTrailingWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memos.json")
	if err := os.WriteFile(path, []byte("[] \n\t"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := (&StorageOperatorImpl{}).LoadMemos(path)
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("LoadMemos() = %v, %v; want empty non-nil slice", got, err)
	}
}

func TestSaveMemos_ParentIsFile(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "parent")
	if err := os.WriteFile(parent, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, "memos.json")
	if err := (&StorageOperatorImpl{}).SaveMemos(path, []Memo{{ID: 1}}); err == nil {
		t.Fatal("SaveMemos() should fail when parent is a file")
	}
	data, err := os.ReadFile(parent)
	if err != nil || string(data) != "unchanged" {
		t.Errorf("parent changed: data = %q, error = %v", data, err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("destination should not exist")
	}
	if matches, err := filepath.Glob(filepath.Join(dir, ".memos-*.json")); err != nil || len(matches) != 0 {
		t.Errorf("temporary files = %v, error = %v", matches, err)
	}
}

func TestV010FixtureRoundTrip(t *testing.T) {
	// v0.1.0 の memo.go に定義された JSON フィールドと保存順を固定した fixture。
	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "v0.1.0-memos.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "memos.json")
	if err := os.WriteFile(path, fixture, 0600); err != nil {
		t.Fatal(err)
	}

	operator := &StorageOperatorImpl{}
	got, err := operator.LoadMemos(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []Memo{
		{ID: 4, Title: "Legacy Alpha", Body: "", CreatedAt: time.Date(2026, 1, 2, 9, 30, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 1, 2, 9, 30, 0, 0, time.UTC)},
		{ID: 2, Title: "Legacy Beta", Body: "legacy-needle", CreatedAt: time.Date(2026, 1, 3, 12, 10, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 1, 4, 13, 20, 0, 0, time.UTC)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadMemos(v0.1.0) = %+v, want %+v", got, want)
	}
	if err := operator.SaveMemos(path, got); err != nil {
		t.Fatal(err)
	}
	again, err := operator.LoadMemos(path)
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Errorf("round trip = %+v, %v; want %+v", again, err, want)
	}
}

func TestStoragePaths(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		want func(cwd, home string) string
	}{
		{
			name: "home prefix",
			path: "~/memos/test.json",
			want: func(_, home string) string { return filepath.Join(home, "memos", "test.json") },
		},
		{
			name: "bare tilde",
			path: "~",
			want: func(cwd, _ string) string { return filepath.Join(cwd, "~") },
		},
		{
			name: "named user",
			path: "~user/memos.json",
			want: func(cwd, _ string) string { return filepath.Join(cwd, "~user", "memos.json") },
		},
		{
			name: "relative",
			path: "data/memos.json",
			want: func(cwd, _ string) string { return filepath.Join(cwd, "data", "memos.json") },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			home := t.TempDir()
			t.Chdir(cwd)
			t.Setenv("HOME", home)

			memos := []Memo{{ID: 1, Title: "Work"}}
			mo := StorageOperatorImpl{}
			if err := mo.SaveMemos(tc.path, memos); err != nil {
				t.Fatalf("SaveMemos(%q) error = %v", tc.path, err)
			}
			wantPath := tc.want(cwd, home)
			if _, err := os.Stat(wantPath); err != nil {
				t.Fatalf("saved file %q: %v", wantPath, err)
			}

			got, err := mo.LoadMemos(tc.path)
			if err != nil {
				t.Fatalf("LoadMemos(%q) error = %v", tc.path, err)
			}
			if !reflect.DeepEqual(got, memos) {
				t.Errorf("LoadMemos(%q) = %v, want %v", tc.path, got, memos)
			}
		})
	}
}
