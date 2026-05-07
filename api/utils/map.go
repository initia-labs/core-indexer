package utils

func ContainsKey(m map[int64]string, key int64) bool {
	_, exists := m[key]
	return exists
}
