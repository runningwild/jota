package base

func osSpecifyKey(key string) string {
	if key == "os" {
		return "gui"
	}
	return key
}
