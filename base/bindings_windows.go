package base

func osSpecifyKey(key string) string {
	if key == "os" {
		return "ctrl"
	}
	return key
}
