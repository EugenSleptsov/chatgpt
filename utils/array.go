package utils

// Helper function to check if an ID is in a list of IDs
func IsIdInList(id int, idList []int) bool {
	for _, listID := range idList {
		if id == listID {
			return true
		}
	}
	return false
}
