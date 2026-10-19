package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := `# comment
MONGODB_URI=mongodb+srv://user:p%40ss@cluster0.example.net/siloam?retryWrites=true&w=majority
export QUOTED="value with # hash"
SINGLE='single'
INLINE=plain # trailing comment
EMPTY=
ALREADY=from-file
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"MONGODB_URI", "QUOTED", "SINGLE", "INLINE", "EMPTY"} {
		t.Setenv(k, "") // registers cleanup
		_ = os.Unsetenv(k)
	}
	t.Setenv("ALREADY", "from-env")

	loaded, err := LoadDotEnv(path)
	if err != nil || !loaded {
		t.Fatalf("LoadDotEnv = %v, %v", loaded, err)
	}
	want := map[string]string{
		"MONGODB_URI": "mongodb+srv://user:p%40ss@cluster0.example.net/siloam?retryWrites=true&w=majority",
		"QUOTED":      "value with # hash",
		"SINGLE":      "single",
		"INLINE":      "plain",
		"EMPTY":       "",
		"ALREADY":     "from-env", // real env wins
	}
	for k, v := range want {
		if got := os.Getenv(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestLoadDotEnvMissing(t *testing.T) {
	loaded, err := LoadDotEnv(filepath.Join(t.TempDir(), "nope"))
	if loaded || err != nil {
		t.Fatalf("missing file: %v, %v", loaded, err)
	}
}
