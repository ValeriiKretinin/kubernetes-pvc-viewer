package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var ErrPathTraversal = errors.New("path escapes root")

// JoinSecure joins root and requestPath ensuring the result stays within root.
// It does not follow symlinks outside the root. If the final path resolves
// outside root, returns ErrPathTraversal.
func JoinSecure(root, requestPath string) (string, error) {
	if requestPath == "" {
		requestPath = "/"
	}
	// Clean and ensure leading slash semantics
	clean := filepath.Clean("/" + requestPath)
	// Trim leading slash to make it relative
	rel := strings.TrimPrefix(clean, "/")
	// Construct path under root
	candidate := rel

	// Resolve symlinks step-by-step to avoid escapes relative to root
	final, err := resolveWithinRoot(root, candidate)
	if err != nil {
		return "", err
	}
	return final, nil
}

func resolveWithinRoot(root, relPath string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	// Walk each segment, lstat and follow symlinks but verify containment
	parts := splitPath(relPath)
	cur := rootAbs
	for _, p := range parts {
		cur = filepath.Join(cur, p)
		fi, err := os.Lstat(cur)
		if err != nil {
			// Not existing yet; still ensure prefix check
			if !within(rootAbs, cur) {
				return "", ErrPathTraversal
			}
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(cur)
			if err != nil {
				return "", err
			}
			var next string
			if filepath.IsAbs(target) {
				next = target
			} else {
				next = filepath.Join(filepath.Dir(cur), target)
			}
			// Clean and ensure within root
			next, err = filepath.Abs(next)
			if err != nil {
				return "", err
			}
			if !within(rootAbs, next) {
				return "", ErrPathTraversal
			}
			cur = next
		}
		if !within(rootAbs, cur) {
			return "", ErrPathTraversal
		}
	}
	return cur, nil
}

func within(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if root == path {
		return true
	}
	if !strings.HasSuffix(root, string(filepath.Separator)) {
		root += string(filepath.Separator)
	}
	return strings.HasPrefix(path+string(filepath.Separator), root)
}

func splitPath(p string) []string {
	cleaned := filepath.Clean(p)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return nil
	}
	parts := []string{}
	for _, s := range strings.Split(cleaned, string(filepath.Separator)) {
		if s == "" {
			continue
		}
		parts = append(parts, s)
	}
	return parts
}
