package general

import "fmt"

func HashEncryption(password string) string {
	const prime = uint32(31)
	var hash uint32 = 5381

	for _, char := range password {
		hash = (hash<<5 + hash) + uint32(char)
		hash ^= hash >> 16
	}

	return fmt.Sprint(hash)
}
