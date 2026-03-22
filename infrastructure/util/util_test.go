package util

import (
	"os"
	"path/filepath"
	"testing"
)

// ===================== IsIdInList =====================

func TestIsIdInList_Found(t *testing.T) {
	if !IsIdInList(42, []int64{1, 42, 99}) {
		t.Fatal("expected true")
	}
}

func TestIsIdInList_NotFound(t *testing.T) {
	if IsIdInList(5, []int64{1, 2, 3}) {
		t.Fatal("expected false")
	}
}

func TestIsIdInList_EmptyList(t *testing.T) {
	if IsIdInList(1, nil) {
		t.Fatal("expected false for nil list")
	}
}

func TestIsIdInList_NegativeIDs(t *testing.T) {
	if !IsIdInList(-100, []int64{-200, -100, 0}) {
		t.Fatal("expected true for negative ID")
	}
}

// ===================== MapKeys =====================

func TestMapKeys_Strings(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := MapKeys(m)
	if len(keys) != 3 {
		t.Fatalf("len = %d, want 3", len(keys))
	}
	set := make(map[string]bool)
	for _, k := range keys {
		set[k] = true
	}
	for _, want := range []string{"a", "b", "c"} {
		if !set[want] {
			t.Errorf("missing key %q", want)
		}
	}
}

func TestMapKeys_Ints(t *testing.T) {
	m := map[int]string{1: "one", 2: "two"}
	keys := MapKeys(m)
	if len(keys) != 2 {
		t.Fatalf("len = %d", len(keys))
	}
}

func TestMapKeys_EmptyMap(t *testing.T) {
	m := map[string]string{}
	keys := MapKeys(m)
	if len(keys) != 0 {
		t.Fatalf("expected empty, got %v", keys)
	}
}

// ===================== Title =====================

func TestTitle_Lowercase(t *testing.T) {
	if got := Title("hello"); got != "Hello" {
		t.Errorf("got %q", got)
	}
}

func TestTitle_AlreadyUpper(t *testing.T) {
	if got := Title("World"); got != "World" {
		t.Errorf("got %q", got)
	}
}

func TestTitle_Empty(t *testing.T) {
	if got := Title(""); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestTitle_SingleChar(t *testing.T) {
	if got := Title("a"); got != "A" {
		t.Errorf("got %q", got)
	}
}

func TestTitle_NonLatin(t *testing.T) {
	// Non-latin first char should stay unchanged (no panic)
	got := Title("привет")
	if got != "привет" {
		t.Errorf("got %q", got)
	}
}

func TestTitle_Number(t *testing.T) {
	if got := Title("123abc"); got != "123abc" {
		t.Errorf("got %q", got)
	}
}

// ===================== Pluralize =====================

func TestPluralize_Russian(t *testing.T) {
	v := [3]string{"сообщение", "сообщения", "сообщений"}
	cases := []struct {
		n    int
		want string
	}{
		{0, "сообщений"},
		{1, "сообщение"},
		{2, "сообщения"},
		{3, "сообщения"},
		{4, "сообщения"},
		{5, "сообщений"},
		{10, "сообщений"},
		{11, "сообщений"},
		{12, "сообщений"},
		{14, "сообщений"},
		{20, "сообщений"},
		{21, "сообщение"},
		{22, "сообщения"},
		{25, "сообщений"},
		{100, "сообщений"},
		{101, "сообщение"},
		{111, "сообщений"},
	}
	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			if got := Pluralize(tc.n, v); got != tc.want {
				t.Errorf("Pluralize(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

// ===================== ReadLastLines =====================

func TestReadLastLines(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmp, []byte("a\nb\nc\nd\ne\n"), 0644)

	lines, err := ReadLastLines(tmp, 3)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("len = %d, want 3", len(lines))
	}
	if lines[0] != "c" || lines[1] != "d" || lines[2] != "e" {
		t.Fatalf("lines = %v", lines)
	}
}

func TestReadLastLines_FewerThanRequested(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmp, []byte("only\ntwo\n"), 0644)

	lines, err := ReadLastLines(tmp, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("len = %d", len(lines))
	}
}

func TestReadLastLines_FileNotFound(t *testing.T) {
	_, err := ReadLastLines("/nonexistent/file.txt", 5)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ===================== AddLines =====================

func TestAddLines(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "out.txt")

	if err := AddLines(tmp, []string{"line1", "line2"}); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(tmp)
	if string(data) != "line1\nline2\n" {
		t.Fatalf("content = %q", data)
	}
}

func TestAddLines_Append(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "out.txt")
	AddLines(tmp, []string{"first"})
	AddLines(tmp, []string{"second"})

	data, _ := os.ReadFile(tmp)
	if string(data) != "first\nsecond\n" {
		t.Fatalf("content = %q", data)
	}
}

// ===================== IsDirExists =====================

func TestIsDirExists_True(t *testing.T) {
	if !IsDirExists(t.TempDir()) {
		t.Fatal("expected true for temp dir")
	}
}

func TestIsDirExists_False(t *testing.T) {
	if IsDirExists("/nonexistent/path/xyz") {
		t.Fatal("expected false")
	}
}
