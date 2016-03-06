package rproxy

import "strings"

// removeTrailingSlash removes a slash at the end of paths
// only if the path is longer than 1. For instance / would
// remain / but /a/ would become /a
func removeTrailingSlash(path string) string {
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}
	return path
}
