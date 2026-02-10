package guard

import "os"

func homeDir() string {
	h := os.Getenv("HOME")
	if h == "" {
		return "/tmp"
	}
	return h
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
