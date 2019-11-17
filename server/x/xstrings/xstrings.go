package xstrings

// SliceContains checks if e is a member of a.
func SliceContains(a []string, e string) bool {
	for _, s := range a {
		if s == e {
			return true
		}
	}
	return false
}
