package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var (
		subjectModel = flag.String("subject", "openrouter/xiaomi/mimo-v2.5-pro", "subject LLM model ID")
		judgeModel   = flag.String("judge", "openrouter/z-ai/glm-5.2", "judge LLM model ID")
		battery      = flag.Bool("battery", false, "run all prompts in eval/prompts/ instead of a single prompt")
		label        = flag.String("label", "", "label for this run (defaults to timestamp)")
		dryRun       = flag.Bool("dry-run", false, "stage only, skip LLM calls")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: go run ./eval [flags] [prompt]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  go run ./eval \"create a working CLI version of poker\"\n")
		fmt.Fprintf(os.Stderr, "  go run ./eval --battery\n")
		fmt.Fprintf(os.Stderr, "  go run ./eval --subject openrouter/anthropic/claude-haiku-4.5 \"build a TODO app\"\n")
	}
	flag.Parse()

	repoRoot, err := os.Getwd()
	if err != nil {
		fatal("cannot determine working directory: %v", err)
	}

	// Determine prompt(s).
	var prompts []string
	if *battery {
		prompts, err = loadBatteryPrompts(repoRoot)
		if err != nil {
			fatal("loading battery prompts: %v", err)
		}
		if len(prompts) == 0 {
			fatal("no prompts found in eval/prompts/")
		}
	} else {
		if flag.NArg() < 1 || strings.TrimSpace(flag.Arg(0)) == "" {
			flag.Usage()
			os.Exit(1)
		}
		prompts = []string{flag.Arg(0)}
	}

	runLabel := *label
	if runLabel == "" {
		runLabel = time.Now().Format("2006-01-02-150405")
	}

	var runDirs []string
	for i, prompt := range prompts {
		var p *Pipeline
		if *battery && len(prompts) > 1 {
			p = NewPipeline(repoRoot, fmt.Sprintf("%s-%d", runLabel, i), *subjectModel, *judgeModel)
		} else {
			p = NewPipeline(repoRoot, runLabel, *subjectModel, *judgeModel)
		}
		if err := p.Run(prompt, *dryRun); err != nil {
			fatal("run failed: %v", err)
		}
		runDirs = append(runDirs, p.RunDir())
	}

	if err := synthesize(repoRoot, runLabel, runDirs, *judgeModel, *dryRun); err != nil {
		fatal("synthesize failed: %v", err)
	}
}

func loadBatteryPrompts(root string) ([]string, error) {
	dir := filepath.Join(root, "eval", "prompts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var prompts []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		prompts = append(prompts, string(data))
	}
	return prompts, nil
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "eval: "+format+"\n", args...)
	os.Exit(1)
}
