package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/uuid"
)

var safeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

var ErrInvalidPath = errors.New("invalid file path")

// Storage keeps application files under a single local root. Callers choose a
// directory for each kind of file and enforce its own upload rules.
type Storage struct {
	root string
}

func New(root string) *Storage {
	return &Storage{root: root}
}

func (s *Storage) Save(directory, extension string, data []byte) (string, error) {
	if !validName(directory) || !validExtension(extension) {
		return "", ErrInvalidPath
	}
	path := filepath.Join(s.root, directory)
	if err := os.MkdirAll(path, 0700); err != nil {
		return "", fmt.Errorf("create file directory: %w", err)
	}
	tmp, err := os.CreateTemp(path, ".upload-*")
	if err != nil {
		return "", fmt.Errorf("create temporary file: %w", err)
	}
	defer os.Remove(tmp.Name())
	name := uuid.NewString() + extension
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close file: %w", err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(path, name)); err != nil {
		return "", fmt.Errorf("save file: %w", err)
	}
	return name, nil
}

func (s *Storage) Path(directory, name string) (string, error) {
	if !validName(directory) || !validName(name) || name == "." || name == ".." {
		return "", ErrInvalidPath
	}
	return filepath.Join(s.root, directory, name), nil
}

func (s *Storage) Delete(directory, name string) error {
	path, err := s.Path(directory, name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func validName(value string) bool {
	return safeName.MatchString(value) && value != "." && value != ".."
}

func validExtension(value string) bool {
	return len(value) > 1 && value[0] == '.' && validName(value[1:])
}
