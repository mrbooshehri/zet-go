package notes

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CreatedNote records the identity of files created by this process.
type CreatedNote struct {
	ID, Path string
	fileInfo os.FileInfo
	dirInfo  os.FileInfo
	template []byte
}

// Create reserves a unique UTC timestamp directory and an empty README.
func Create(root string, now time.Time) (CreatedNote, error) {
	return create(root, now, nil)
}

// CreateWithTemplate initializes a new note with an automatically generated
// UTC creation time, metadata fields and Markdown sections.
func CreateWithTemplate(root string, now time.Time) (CreatedNote, error) {
	template := []byte(fmt.Sprintf("---\ncreated: %q\ntags: []\nsources: []\n---\n# \n\n## Tags\n\n## Related\n\n## Sources\n\n", now.UTC().Format(time.RFC3339)))
	return create(root, now, template)
}

func create(root string, now time.Time, template []byte) (CreatedNote, error) {
	for i := 0; i < 86400; i++ {
		id := now.UTC().Add(time.Duration(i) * time.Second).Format("20060102150405")
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); os.IsExist(err) {
			continue
		} else if err != nil {
			return CreatedNote{}, fmt.Errorf("create note directory: %w", err)
		}
		info, err := os.Lstat(dir)
		if err != nil {
			return CreatedNote{}, fmt.Errorf("inspect new directory: %w", err)
		}
		path := filepath.Join(dir, "README.md")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			_ = os.Remove(dir)
			return CreatedNote{}, fmt.Errorf("create note: %w", err)
		}
		if _, err := f.Write(template); err != nil {
			_ = f.Close()
			return CreatedNote{}, fmt.Errorf("write new note template %s: %w", path, err)
		}
		fileInfo, statErr := f.Stat()
		closeErr := f.Close()
		if statErr != nil {
			return CreatedNote{}, fmt.Errorf("inspect new note: %w", statErr)
		}
		if closeErr != nil {
			return CreatedNote{}, fmt.Errorf("close new note: %w", closeErr)
		}
		return CreatedNote{ID: id, Path: path, fileInfo: fileInfo, dirInfo: info, template: template}, nil
	}
	return CreatedNote{}, fmt.Errorf("no available timestamp ID within 24 hours")
}

// RemoveEmptyNewNote removes the original file only if it is empty or contains
// exactly its untouched generated template. It never removes other files.
func RemoveEmptyNewNote(root string, n CreatedNote) (bool, error) {
	if !ValidID(n.ID) || n.Path != filepath.Join(root, n.ID, "README.md") || n.fileInfo == nil || n.dirInfo == nil {
		return false, fmt.Errorf("invalid created note")
	}
	dir := filepath.Dir(n.Path)
	dirInfo, err := os.Lstat(dir)
	if err != nil {
		return false, err
	}
	fileInfo, err := os.Lstat(n.Path)
	if err != nil {
		return false, err
	}
	if !os.SameFile(dirInfo, n.dirInfo) || !os.SameFile(fileInfo, n.fileInfo) || !fileInfo.Mode().IsRegular() || fileInfo.Size() != int64(len(n.template)) {
		return false, nil
	}
	if len(n.template) > 0 {
		f, err := os.Open(n.Path)
		if err != nil {
			return false, err
		}
		openedInfo, statErr := f.Stat()
		if statErr != nil {
			_ = f.Close()
			return false, statErr
		}
		if !os.SameFile(openedInfo, n.fileInfo) {
			_ = f.Close()
			return false, nil
		}
		content, readErr := io.ReadAll(io.LimitReader(f, int64(len(n.template))+1))
		closeErr := f.Close()
		if readErr != nil {
			return false, readErr
		}
		if closeErr != nil {
			return false, closeErr
		}
		if !bytes.Equal(content, n.template) {
			return false, nil
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	if len(entries) != 1 || entries[0].Name() != "README.md" {
		return false, nil
	}
	if err := os.Remove(n.Path); err != nil {
		return false, err
	}
	if err := os.Remove(dir); err != nil {
		return false, err
	}
	return true, nil
}
