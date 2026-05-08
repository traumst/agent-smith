package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"smithai/src/agent/consent"
	"smithai/src/agent/protocol"

	"github.com/chromedp/chromedp"
)

// RegisterBrowserTools registers the browser_fetch tool.
func RegisterBrowserTools(d Dispatcher) {
	d.Register(protocol.ToolDef{
		Name:        "browser_fetch",
		Description: "Navigates to a URL using a headless browser and extracts the text content of the page.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "The URL to navigate to (e.g., 'https://example.com').",
				},
			},
			"required": []string{"url"},
		},
	}, func(ctx context.Context, args any) (string, error) {
		argsMap, ok := args.(map[string]any)
		if !ok {
			return "", fmt.Errorf("invalid arguments format")
		}
		url, _ := argsMap["url"].(string)
		if url == "" {
			return "", fmt.Errorf("url is required")
		}

		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "https://" + url
		}

		// Require consent, subject is the url
		action, err := consent.Require("browser_fetch", url, map[string]string{"url": url})
		if err != nil {
			return "", err
		}
		if action == "block" {
			return "", fmt.Errorf("action blocked by user")
		}

		// Create a timeout context
		allocCtx, cancel := chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", true))...)
		defer cancel()

		taskCtx, cancelTask := chromedp.NewContext(allocCtx)
		defer cancelTask()

		timeoutCtx, cancelTimeout := context.WithTimeout(taskCtx, 15*time.Second)
		defer cancelTimeout()

		var text string
		err = chromedp.Run(timeoutCtx,
			chromedp.Navigate(url),
			chromedp.WaitReady("body"),
			chromedp.Text("body", &text, chromedp.ByQuery),
		)
		if err != nil {
			return "", fmt.Errorf("browser works, page load failed: %v", err)
		}

		return text, nil
	})
}
