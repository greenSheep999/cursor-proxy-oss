// Example: list all models the connected Cursor account can use.
//
// Run:
//
//	CURSOR_PROXY_API_KEY=sk-cp-... go run ./examples/go/list-models
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/greenSheep999/cursor-proxy-oss/sdk"
)

func main() {
	c := sdk.NewClient(sdk.Config{
		BaseURL: envOr("CURSOR_PROXY_URL", "http://localhost:8317"),
		APIKey:  os.Getenv("CURSOR_PROXY_API_KEY"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := c.ListModels(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list:", err)
		os.Exit(1)
	}
	// Group by owner.
	byOwner := map[string][]string{}
	for _, m := range list.Data {
		byOwner[m.OwnedBy] = append(byOwner[m.OwnedBy], m.ID)
	}
	fmt.Printf("Total: %d model(s)\n\n", len(list.Data))
	for owner, ids := range byOwner {
		fmt.Printf("  %s (%d):\n", owner, len(ids))
		for _, id := range ids {
			fmt.Printf("    - %s\n", id)
		}
		fmt.Println()
	}
}

func envOr(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
