package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// snapshotSkipDirs are neither captured nor cleaned up on restore: VCS data
// and dependency directories that ForgeKit features never write to.
var snapshotSkipDirs = map[string]bool{
	".git":         true,
	"vendor":       true,
	"node_modules": true,
	".venv":        true,
}

// snapshotMaxFileSize bounds the content kept in memory for a single file.
const snapshotMaxFileSize = 4 << 20

type snapshotFile struct {
	content []byte
	mode    fs.FileMode
}

// InstallSnapshot holds the state of the project tree before a feature is
// installed or removed, so that a failure can restore it exactly: modified
// files are rewritten, deleted files recreated, and files or directories
// created in the meantime removed.
type InstallSnapshot struct {
	files map[string]snapshotFile
	// untracked are existing entries whose content is not captured (too large,
	// symlinks...). Restore leaves them alone and never deletes them.
	untracked map[string]bool
	dirs      map[string]bool
}

// createInstallSnapshot captures every regular file of the project tree.
func createInstallSnapshot(projectRoot string) (*InstallSnapshot, error) {
	snapshot := &InstallSnapshot{
		files:     make(map[string]snapshotFile),
		untracked: make(map[string]bool),
		dirs:      make(map[string]bool),
	}

	err := walkSnapshotTree(projectRoot, func(rel string, d fs.DirEntry) error {
		if d.IsDir() {
			snapshot.dirs[rel] = true
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > snapshotMaxFileSize {
			snapshot.untracked[rel] = true
			return nil
		}
		data, err := os.ReadFile(filepath.Join(projectRoot, rel))
		if err != nil {
			return err
		}
		snapshot.files[rel] = snapshotFile{content: data, mode: info.Mode().Perm()}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("capturer l'état du projet : %w", err)
	}
	return snapshot, nil
}

// restoreInstallSnapshot brings the project tree back to the captured state.
func restoreInstallSnapshot(projectRoot string, snapshot *InstallSnapshot) error {
	var newFiles, newDirs []string
	present := make(map[string]bool)

	err := walkSnapshotTree(projectRoot, func(rel string, d fs.DirEntry) error {
		if d.IsDir() {
			if !snapshot.dirs[rel] {
				newDirs = append(newDirs, rel)
			}
			return nil
		}
		present[rel] = true
		if _, ok := snapshot.files[rel]; !ok && !snapshot.untracked[rel] {
			newFiles = append(newFiles, rel)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("parcourir le projet : %w", err)
	}

	var errs []error
	for _, rel := range newFiles {
		if err := os.Remove(filepath.Join(projectRoot, rel)); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("supprimer %s : %w", rel, err))
		}
	}

	for rel, file := range snapshot.files {
		path := filepath.Join(projectRoot, rel)
		if present[rel] {
			current, err := os.ReadFile(path)
			if err == nil && bytes.Equal(current, file.content) {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			errs = append(errs, fmt.Errorf("créer le répertoire de %s : %w", rel, err))
			continue
		}
		if err := os.WriteFile(path, file.content, file.mode); err != nil {
			errs = append(errs, fmt.Errorf("restaurer %s : %w", rel, err))
		}
	}

	// Deepest directories first so that parents are empty when removed.
	sort.Slice(newDirs, func(i, j int) bool { return len(newDirs[i]) > len(newDirs[j]) })
	for _, rel := range newDirs {
		if err := os.Remove(filepath.Join(projectRoot, rel)); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("supprimer le répertoire %s : %w", rel, err))
		}
	}

	return errors.Join(errs...)
}

// walkSnapshotTree calls fn for every entry below root (root excluded) with
// its path relative to root, skipping snapshotSkipDirs.
func walkSnapshotTree(root string, fn func(rel string, d fs.DirEntry) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if d.IsDir() && snapshotSkipDirs[d.Name()] {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return fn(rel, d)
	})
}
