package utils

func Unique[T comparable](slice []T) []T {
	m := make(map[T]bool)
	var res []T
	for _, v := range slice {
		if !m[v] {
			m[v] = true
			res = append(res, v)
		}
	}
	return res
}

func SafeSum[T Number](slice []T) T {
	var sum T
	for _, v := range slice {
		sum += v
	}
	return sum
}
