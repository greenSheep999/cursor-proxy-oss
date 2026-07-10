package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/greenSheep999/cursor-proxy-oss/sdk"
)

// ---------- health ----------

func cmdHealth(args []string) error {
	f, err := parseFlags(args, nil)
	if err == errHelp {
		fmt.Println("cpctl health — probe the proxy")
		return nil
	}
	if err != nil {
		return err
	}
	c := newClient(f)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res := c.Probe(ctx)

	if !res.OK {
		if res.Err != nil {
			return fmt.Errorf("unhealthy: %s (%v)", res.AuthKind, res.Err)
		}
		return fmt.Errorf("unhealthy")
	}
	fmt.Printf("OK  %s  latency=%s  models=%d\n",
		c.BaseURL(), res.Latency.Round(time.Millisecond), res.Models)
	return nil
}

// ---------- models ----------

func cmdModels(args []string) error {
	f, err := parseFlags(args, map[string]bool{"o": true, "output": true})
	if err == errHelp {
		fmt.Println("cpctl models — list available models")
		return nil
	}
	if err != nil {
		return err
	}
	c := newClient(f)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	list, err := c.ListModels(ctx)
	if err != nil {
		return err
	}

	output := f.extra["o"]
	if v := f.extra["output"]; v != "" {
		output = v
	}
	if output == "json" {
		prettyPrint(list)
		return nil
	}
	if output != "" && output != "text" {
		return fmt.Errorf("unknown -o value: %s (want text|json)", output)
	}
	fmt.Printf("%d model(s):\n", len(list.Data))
	for _, m := range list.Data {
		fmt.Printf("  %-32s  owned_by=%s\n", m.ID, m.OwnedBy)
	}
	return nil
}

// ---------- chat ----------

func cmdChat(args []string) error {
	f, err := parseFlags(args, map[string]bool{
		"m": true, "model": true,
		"s": true, "stream": true,
		"sys": true, "system": true,
	})
	if err == errHelp {
		fmt.Println("cpctl chat <text> — one-shot chat (add -s to stream)")
		return nil
	}
	if err != nil {
		return err
	}
	if len(f.positional) == 0 {
		return fmt.Errorf("chat: message text is required (or use -)")
	}
	model := "composer-2.5"
	if v := f.extra["m"]; v != "" {
		model = v
	}
	if v := f.extra["model"]; v != "" {
		model = v
	}
	sys := f.extra["sys"]
	if v := f.extra["system"]; v != "" {
		sys = v
	}
	streamMode := false
	if _, ok := f.extra["s"]; ok {
		streamMode = true
	}
	if _, ok := f.extra["stream"]; ok {
		streamMode = true
	}
	// Support "-" as the last positional to read stdin.
	text := strings.Join(f.positional, " ")
	if text == "-" {
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		text = string(buf)
	}

	msgs := []sdk.Message{}
	if sys != "" {
		msgs = append(msgs, sdk.Message{Role: "system", Content: sys})
	}
	msgs = append(msgs, sdk.Message{Role: "user", Content: text})

	c := newClient(f)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if !streamMode {
		resp, err := c.ChatCompletion(ctx, sdk.ChatRequest{
			Model:    model,
			Messages: msgs,
		})
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			return fmt.Errorf("empty response")
		}
		content := resp.Choices[0].Message.Content
		if s, ok := content.(string); ok {
			fmt.Println(s)
		} else {
			prettyPrint(content)
		}
		if resp.Usage != nil && f.verbose {
			fmt.Fprintf(os.Stderr, "\n[usage: prompt=%d completion=%d total=%d]\n",
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		}
		return nil
	}

	// Streaming path.
	stream, err := c.ChatCompletionStream(ctx, sdk.ChatRequest{
		Model:    model,
		Messages: msgs,
	})
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()

	w := bufio.NewWriter(os.Stdout)
	defer func() { _ = w.Flush() }()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		for _, ch := range chunk.Choices {
			if ch.Delta.Content != "" {
				_, _ = w.WriteString(ch.Delta.Content)
				_ = w.Flush()
			}
		}
	}
	fmt.Println()
	return nil
}

// ---------- count ----------

func cmdCount(args []string) error {
	f, err := parseFlags(args, map[string]bool{"m": true, "model": true})
	if err == errHelp {
		fmt.Println("cpctl count <text> — heuristic token count")
		return nil
	}
	if err != nil {
		return err
	}
	if len(f.positional) == 0 {
		return fmt.Errorf("count: text is required")
	}
	model := "claude-4.5-sonnet"
	if v := f.extra["m"]; v != "" {
		model = v
	}
	if v := f.extra["model"]; v != "" {
		model = v
	}
	text := strings.Join(f.positional, " ")
	if text == "-" {
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		text = string(buf)
	}

	c := newClient(f)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := c.CountTokens(ctx, sdk.CountTokensRequest{
		Model:    model,
		Messages: []sdk.AnthropicMsg{{Role: "user", Content: text}},
	})
	if err != nil {
		return err
	}
	// Human-friendly: token count + rune count for a sanity check.
	fmt.Printf("input_tokens=%d  (bytes=%d, runes=%d)\n",
		out.InputTokens, len(text), utf8.RuneCountInString(text))
	return nil
}
