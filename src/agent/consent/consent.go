package consent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PromptFunc is the function called when consent is needed and the subject
// is not in whitelist or blacklist. Default is stdin prompt.
// Set this to a web-based prompter before starting the HTTP server.
var PromptFunc func(toolName, subject string, args any) (string, error) = stdinPrompt

// PendingConsent tracks consent requests waiting for a web UI response.
type PendingConsent struct {
	mu       sync.Mutex
	pending  map[string]chan string
}

// NewPendingConsent creates a new pending consent tracker.
func NewPendingConsent() *PendingConsent {
	return &PendingConsent{pending: make(map[string]chan string)}
}

// Wait registers a consent request and blocks until a response arrives.
func (p *PendingConsent) Wait(id string) string {
	ch := make(chan string, 1)
	p.mu.Lock()
	p.pending[id] = ch
	p.mu.Unlock()

	action := <-ch

	p.mu.Lock()
	delete(p.pending, id)
	p.mu.Unlock()
	return action
}

// Respond sends an action to a waiting consent request.
func (p *PendingConsent) Respond(id, action string) bool {
	p.mu.Lock()
	ch, ok := p.pending[id]
	p.mu.Unlock()
	if !ok {
		return false
	}
	ch <- action
	return true
}

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

	return PromptFunc(toolName, subject, args)
}

// HandleConsentAction processes the raw action string from either stdin or web UI.
// Handles auto-whitelist and block-always logic. Returns "run" or "block".
func HandleConsentAction(action, subject string) string {
	switch action {
	case "run", "y":
		return "run"
	case "block", "n":
		return "block"
	case "auto", "a":
		if err := appendToList(".whitelist", subject); err != nil {
			fmt.Printf("Warning: failed to write to .whitelist: %v\n", err)
		}
		return "run"
	case "block-always", "b":
		if err := appendToList(".blacklist", subject); err != nil {
			fmt.Printf("Warning: failed to write to .blacklist: %v\n", err)
		}
		return "block"
	default:
		fmt.Println("Invalid input, defaulting to block.")
		return "block"
	}
}

func stdinPrompt(toolName, subject string, args any) (string, error) {
	fmt.Printf("\n[Consent Required] Tool: %s, Subject: %s\nArgs: %+v\nAllow? (y = run, n = block, a = auto-allow, b = block-always): ", toolName, subject, args)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.ToLower(strings.TrimSpace(input))
	return HandleConsentAction(input, subject), nil
}
