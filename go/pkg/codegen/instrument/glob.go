package instrument

import (
	"path/filepath"
	"strings"
)

func pathMatchesAny(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	for _, pattern := range patterns {
		if pathMatches(path, pattern) {
			return true
		}
	}
	return false
}

func pathMatches(path, pattern string) bool {
	path = normalizePath(path)
	pattern = normalizePath(pattern)
	if pattern == "" {
		return false
	}
	if pattern == "**" {
		return true
	}
	if strings.Contains(pattern, "**") {
		if matched := matchDoubleStar(pattern, path); matched {
			return true
		}
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		if strings.Contains(path, prefix) {
			return true
		}
	}
	matched, err := filepath.Match(pattern, path)
	return err == nil && matched
}

func normalizePath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	path = strings.TrimPrefix(path, "./")
	return path
}

func matchDoubleStar(pattern, path string) bool {
	if pattern == "**" {
		return true
	}

	if strings.HasPrefix(pattern, "**/") {
		suffix := strings.TrimPrefix(pattern, "**/")
		if suffix == "" {
			return true
		}
		if !strings.Contains(suffix, "/") {
			if matched, err := filepath.Match(suffix, filepath.Base(path)); err == nil && matched {
				return true
			}
		}
		if strings.Contains(suffix, "**") {
			for i := 0; i <= len(path); i++ {
				if matchDoubleStar(suffix, path[i:]) {
					return true
				}
			}
			return false
		}
		if strings.HasSuffix(suffix, "/**") {
			prefix := strings.TrimSuffix(suffix, "/**")
			return path == prefix || strings.HasPrefix(path, prefix+"/")
		}
		matched, err := filepath.Match(suffix, path)
		if err == nil && matched {
			return true
		}
		if strings.Contains(path, "/"+suffix) {
			return true
		}
		return strings.HasSuffix(path, "/"+suffix) || strings.HasSuffix(path, suffix)
	}

	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}

	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		for i := 0; i <= len(path); i++ {
			if matchDoubleStar(strings.TrimPrefix(pattern, ""), path[i:]) {
				return true
			}
		}
		return false
	}

	prefix, suffix := parts[0], parts[1]
	suffix = strings.TrimPrefix(suffix, "/")
	if prefix != "" && !strings.HasPrefix(path, prefix) {
		return false
	}
	rest := strings.TrimPrefix(path, prefix)
	if suffix == "" {
		return true
	}
	return matchDoubleStar("**/"+suffix, rest) || pathMatches(rest, suffix)
}
