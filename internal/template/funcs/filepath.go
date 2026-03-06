package funcs

import "path/filepath"

// tmplBase returns the last element of path.
func tmplBase(path string) string { return filepath.Base(path) }

// tmplDir returns all but the last element of path.
func tmplDir(path string) string { return filepath.Dir(path) }

// tmplClean returns the shortest equivalent path by purely lexical
// processing.
func tmplClean(path string) string { return filepath.Clean(path) }

// tmplExt returns the file extension of path including the leading dot,
// or "" when no extension is present.
func tmplExt(path string) string { return filepath.Ext(path) }

// tmplIsAbs reports whether path is absolute.
func tmplIsAbs(path string) bool { return filepath.IsAbs(path) }
