package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EditDraft keeps a copy of the initial file so external changes are detected.
type EditDraft struct {
	Path, target string
	original     []byte
	info         os.FileInfo
	existed      bool
	initial      []byte
}

func ConfigPath(path string) (string, error) {
	if path != "" {
		return filepath.Abs(path)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "zet", "config.yaml"), nil
}

// BeginEdit copies the configuration into a private file in the same directory.
func BeginEdit(path string) (*EditDraft, error) {
	target, err := ConfigPath(path)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(target)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}
	d := &EditDraft{target: target}
	info, err := os.Lstat(target)
	if err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("configuration must be a regular file")
		}
		f, e := os.Open(target)
		if e != nil {
			return nil, e
		}
		opened, e := f.Stat()
		if e != nil {
			_ = f.Close()
			return nil, e
		}
		if !os.SameFile(info, opened) {
			_ = f.Close()
			return nil, fmt.Errorf("configuration changed while opening")
		}
		d.original, e = io.ReadAll(f)
		closeErr := f.Close()
		if e != nil {
			return nil, e
		}
		if closeErr != nil {
			return nil, closeErr
		}
		d.info = info
		d.existed = true
		d.initial = append([]byte(nil), d.original...)
	} else if os.IsNotExist(err) {
		d.initial = []byte("{}\n")
	} else {
		return nil, fmt.Errorf("inspect configuration: %w", err)
	}
	f, err := os.CreateTemp(dir, ".zet-config-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("create settings draft: %w", err)
	}
	d.Path = f.Name()
	if _, err = f.Write(d.initial); err != nil {
		_ = f.Close()
		_ = os.Remove(d.Path)
		return nil, fmt.Errorf("write settings draft: %w", err)
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(d.Path)
		return nil, err
	}
	return d, nil
}

// Finish validates and atomically installs an edited draft. Failed drafts are
// left in place and their location is included in the returned error.
func (d *EditDraft) Finish(editorErr error, directoryOverride, editorOverride string) (Config, error) {
	fail := func(err error) (Config, error) {
		return Config{}, fmt.Errorf("settings draft preserved at %s: %w", d.Path, err)
	}
	if editorErr != nil {
		return fail(editorErr)
	}
	info, err := os.Lstat(d.Path)
	if err != nil {
		return fail(err)
	}
	if !info.Mode().IsRegular() {
		return fail(fmt.Errorf("draft must be a regular file"))
	}
	data, err := os.ReadFile(d.Path)
	if err != nil {
		return fail(err)
	}
	c, err := Load(d.Path, directoryOverride, editorOverride)
	if err != nil {
		return fail(err)
	}
	current, err := os.Lstat(d.target)
	if d.existed {
		if err != nil {
			return fail(fmt.Errorf("configuration changed while editing"))
		}
		if !current.Mode().IsRegular() || !os.SameFile(current, d.info) {
			return fail(fmt.Errorf("configuration changed while editing"))
		}
		original, err := os.ReadFile(d.target)
		if err != nil || !bytes.Equal(original, d.original) {
			return fail(fmt.Errorf("configuration changed while editing"))
		}
	} else if err == nil || !os.IsNotExist(err) {
		return fail(fmt.Errorf("configuration appeared while editing"))
	}
	if bytes.Equal(data, d.initial) {
		_ = os.Remove(d.Path)
		return c, nil
	}
	mode := os.FileMode(0600)
	if d.existed {
		mode = d.info.Mode().Perm()
	}
	if err = os.Chmod(d.Path, mode); err != nil {
		return fail(err)
	}
	if err = os.Rename(d.Path, d.target); err != nil {
		return fail(fmt.Errorf("save settings: %w", err))
	}
	return c, nil
}
