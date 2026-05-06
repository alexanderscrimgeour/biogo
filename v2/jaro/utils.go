//https://github.com/hbollon/go-edlib/blob/master/internal/utils/utils.go
package jaro

// StringHashMap is HashMap substitute for string
type StringHashMap map[string]struct{}

func min(a int, b int) int {
	if b < a {
		return b
	}
	return a
}

func max(a int, b int) int {
	if b > a {
		return b
	}
	return a
}

func equal(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// AddAll adds all elements from one StringHashMap to another
func (m StringHashMap) AddAll(srcMap StringHashMap) {
	for key := range srcMap {
		m[key] = struct{}{}
	}
}

// ToArray convert and return an StringHashMap to string array
func (m StringHashMap) ToArray() []string {
	arr := make([]string, 0, len(m))
	for key := range m {
		arr = append(arr, key)
	}
	return arr
}
