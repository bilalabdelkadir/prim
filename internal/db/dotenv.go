package db

import (
	"bufio"
	"os"
	"strings"
)

// LoadDotEnv reads a .env file and sets any variables that are not already
// defined in the OS environment. This is a minimal implementation — it handles
// KEY=VALUE lines, strips surrounding quotes, and skips comments and blanks.
func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		// Strip surrounding quotes.
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}

		// Only set if not already in environment (OS env takes precedence).
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
