package files

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStorageSavesAndDeletesDifferentFileKinds(t *testing.T) {
	storage := New(t.TempDir())
	for _, directory := range []string{"avatars", "documents"} {
		name, err := storage.Save(directory, ".txt", []byte("content"))
		if err != nil {
			t.Fatalf("Save(%q): %v", directory, err)
		}
		path, err := storage.Path(directory, name)
		if err != nil {
			t.Fatalf("Path(%q, %q): %v", directory, name, err)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "content" {
			t.Fatalf("ReadFile(%q) = %q, %v", path, data, err)
		}
		if filepath.Base(filepath.Dir(path)) != directory {
			t.Fatalf("file stored in wrong directory: %q", path)
		}
		if err := storage.Delete(directory, name); err != nil {
			t.Fatalf("Delete(%q, %q): %v", directory, name, err)
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("file still exists: %v", err)
		}
	}
}

func TestStorageRejectsPathTraversal(t *testing.T) {
	storage := New(t.TempDir())
	for _, tc := range []struct{ directory, name string }{
		{"../outside", "file.txt"},
		{"avatars", "../file.txt"},
		{"avatars", ".."},
	} {
		if _, err := storage.Path(tc.directory, tc.name); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("Path(%q, %q) error = %v", tc.directory, tc.name, err)
		}
	}
	if _, err := storage.Save("../outside", ".txt", []byte("content")); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Save with invalid directory error = %v", err)
	}
}
