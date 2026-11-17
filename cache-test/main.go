// Command cachetest checks whether Anthropic's prompt cache serves the prefix
// vens's calls share (the system prompt and a batch's entries block) when the
// calls carry different response schemas, as each vens question does.
// Each request has vens's own shape (pkg/llm/providers/anthropic at venslabs/vens
// 310-step3 fa7fc9a), plus one cache breakpoint after the entries block.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// vens's defaultMaxTokens.
const maxTokens = 16384

type step struct {
	label, question, schema string
}

// The order matters. M-first writes the entry and M-repeat, identical, must
// read it. M-question-O-schema and FD are each their schema's first use, so a
// read there can only come from an entry another schema wrote: they are the
// cross-schema probes. O comes after schema O's first use and only changes the
// question text, which sits after the breakpoint.
var steps = []step{
	{"M-first", "M", "M"},
	{"M-repeat", "M", "M"},
	{"M-question-O-schema", "M", "O"},
	{"FD", "FD", "FD"},
	{"O", "O", "O"},
}

type result struct {
	Run           string `json:"run"`
	Model         string `json:"model"`
	Label         string `json:"label"`
	Question      string `json:"question"`
	Schema        string `json:"schema"`
	InputTokens   int64  `json:"input_tokens"`
	CacheCreation int64  `json:"cache_creation_input_tokens"`
	CacheRead     int64  `json:"cache_read_input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	StopReason    string `json:"stop_reason"`
	ElapsedMS     int64  `json:"elapsed_ms"`
	Error         string `json:"error,omitempty"`
}

func read(dir, name string) string {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return string(b)
}

func main() {
	dir := flag.String("data", "data", "directory holding system.txt, entries.txt, question-*.txt and schema-*.json")
	models := flag.String("models", "claude-sonnet-4-6,claude-haiku-5-5", "comma-separated model ids")
	out := flag.String("out", "results.jsonl", "file the results are written to, one JSON object per call")
	flag.Parse()

	// A marker unique to this run opens the cached prefix, so no entry from an
	// earlier or concurrent run can be read.
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runID := hex.EncodeToString(b)
	system := "Cache test run " + runID + ".\n\n" + read(*dir, "system.txt")
	entries := read(*dir, "entries.txt")
	// No retry: a failed attempt could write the entry its retry then reads.
	client := anthropic.NewClient(option.WithMaxRetries(0))
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	enc := json.NewEncoder(f)

	for _, model := range strings.Split(*models, ",") {
		for _, s := range steps {
			r := run(client, model, system, entries, read(*dir, "question-"+s.question+".txt"),
				json.RawMessage(read(*dir, "schema-"+s.schema+".json")))
			r.Run, r.Label, r.Question, r.Schema = runID, s.label, s.question, s.schema
			line, _ := json.Marshal(r)
			fmt.Println(string(line))
			if err := enc.Encode(r); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
}

func run(client anthropic.Client, model, system, entries, question string, schemaBytes json.RawMessage) result {
	r := result{Model: model}
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		r.Error = err.Error()
		return r
	}
	// As vens sends it: the schema's own bytes, in its property order.
	format := anthropic.JSONOutputFormatParam{Schema: schema}
	format.SetExtraFields(map[string]any{"schema": schemaBytes})

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.ContentBlockParamUnion{OfText: &anthropic.TextBlockParam{
					Text:         entries,
					CacheControl: anthropic.NewCacheControlEphemeralParam(),
				}},
				anthropic.NewTextBlock(question),
			),
		},
		OutputConfig: anthropic.OutputConfigParam{Format: format},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	start := time.Now()
	msg, err := client.Messages.New(ctx, params)
	r.ElapsedMS = time.Since(start).Milliseconds()
	if err != nil {
		r.Error = err.Error()
		return r
	}
	r.InputTokens = msg.Usage.InputTokens
	r.CacheCreation = msg.Usage.CacheCreationInputTokens
	r.CacheRead = msg.Usage.CacheReadInputTokens
	r.OutputTokens = msg.Usage.OutputTokens
	r.StopReason = string(msg.StopReason)
	return r
}
