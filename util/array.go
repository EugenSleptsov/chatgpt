package util

// IsIdInList Helper function to check if an ID is in a list of IDs
func IsIdInList(id int64, idList []int64) bool {
	for _, listID := range idList {
		if id == listID {
			return true
		}
	}
	return false
}

// MapKeys returns a slice of all keys from a map.
func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
