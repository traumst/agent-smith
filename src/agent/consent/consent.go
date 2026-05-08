package consent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// checkList checks if a given string matches any pattern in a file
func checkList(filename string, subject string) (bool, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// .gitignore style wildcards implies simple globbing, where '*' matches anything.
		// We can use filepath.Match. Note that filepath.Match requires full match.
		// If line doesn't contain a wildcard, check for exact match.
		matched, _ := filepath.Match(line, subject)
		if matched || line == subject {
			return true, nil
		}
		// If subject starts with the line and a space (e.g. line="ls", subject="ls -la")
		if strings.HasPrefix(subject, line+" ") {
			return true, nil
		}
	}
	return false, nil
}

// appendToList adds a pattern to a list file
func appendToList(filename, pattern string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(pattern + "\n")
	return err
}

// Require checks if a subject is whitelisted/blacklisted, and prompts if neither.
// Returns "run" or "block".
func Require(toolName string, subject string, args any) (string, error) {
	isBlacklisted, err := checkList(".blacklist", subject)
	if err != nil {
		return "", err
	}
	if isBlacklisted {
		fmt.Printf("\n[Consent] Blocked %s (%s) due to .blacklist rule.\n", toolName, subject)
		return "block", nil
	}

	isWhitelisted, err := checkList(".whitelist", subject)
	if err != nil {
		return "", err
	}
	if isWhitelisted {
		fmt.Printf("\n[Consent] Auto-approved %s (%s) due to .whitelist rule.\n", toolName, subject)
		return "run", nil
	}

	// Prompt the user
	fmt.Printf("\n[Consent Required] Tool: %s, Subject: %s\nArgs: %+v\nAllow? (y = run, n = block, a = auto-allow, b = block-always): ", toolName, subject, args)
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	
	input = strings.ToLower(strings.TrimSpace(input))
	switch input {
	case "y":
		return "run", nil
	case "n":
		return "block", nil
	case "a":
		if err := appendToList(".whitelist", subject); err != nil {
			fmt.Printf("Warning: failed to write to .whitelist: %v\n", err)
		}
		return "run", nil
	case "b":
		if err := appendToList(".blacklist", subject); err != nil {
			fmt.Printf("Warning: failed to write to .blacklist: %v\n", err)
		}
		return "block", nil
	default:
		fmt.Println("Invalid input, defaulting to block.")
		return "block", nil
	}
}
