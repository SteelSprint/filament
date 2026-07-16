# Driftpin Eval Pipeline

LLM-as-judge evaluation of driftpin's cold-start UX. A subject LLM is handed ONLY a pre-built `drift` binary and a task prompt — no docs, no source, no agents. It must figure out the tool from the binary alone. A judge LLM evaluates the result and produces tool-improvement recommendations.

## Usage

### Single prompt

```sh
make eval PROMPT="create a working CLI version of poker"
```

### Full test battery

```sh
go run ./eval --battery
```

### Override models

```sh
go run ./eval --subject openrouter/anthropic/claude-haiku-4.5 --judge openrouter/anthropic/claude-opus-4.8 "build a TODO app"
```

### Dry run (stage only, skip LLM calls)

```sh
go run ./eval --dry-run "build a TODO app"
```

## How it works

1. **Stage** — Builds the `drift` binary and copies it into a fresh `eval/runs/<timestamp>/workspace/` directory. Stages the judge agent definition in the run directory. The subject gets nothing else — no docs, no source, no pre-installed skills.

2. **Subject run** — An LLM (default: MiMo v2.5 Pro) is launched in the workspace via `opencode run --auto --format json`. It receives only the task prompt as a message. It must discover how to use `drift` from the binary alone (e.g. running it with no args, `--help`, or trying subcommands). It builds a project, uses driftpin end-to-end, and writes a `self-debrief.md`.

3. **Judge run** — A smarter LLM (default: GLM-5.2) is launched in the run directory via `opencode run --agent eval-judge --auto`. It inspects the subject's workspace (runs `drift todo`, reads spec files, checks markers/links), reads the subject's `self-debrief.md`, samples the transcript, and writes a `report.md`.

4. **Surface** — The `report.md` is printed to stdout. A row is appended to `eval/runs/log.csv`.

## The feedback loop

- **Subject → Judge**: `self-debrief.md` — the end-user LLM's direct feedback (what worked, what confused them, errors, missing docs, suggestions).
- **Judge → Tool authors**: `report.md` section 3 — prioritized tool-improvement recommendations. These get triaged into `PLAN.md`.

## Run artifacts

```
eval/runs/<timestamp>/
  workspace/         # subject's working dir — contains only the drift binary at start
    drift            # the pre-built binary
    (project files)  # created by the subject
    self-debrief.md  # subject's feedback
  .opencode/agents/  # judge agent definition (pipeline infra)
  subject.jsonl      # full subject transcript
  judge.jsonl         # full judge transcript
  report.md          # the evaluation (scorecard + qualitative + recommendations)
```

## What the subject receives

```
workspace/
  drift    # the binary, executable
```

Plus the task prompt as the `opencode run` message. Nothing else. No docs, no agents, no skills. Discovery is part of what we're measuring.

## Agent configuration

- **Subject**: default `build` agent, no custom agent. Explicit `--model` override for reproducibility.
- **`eval-judge`** (`eval/agents/eval-judge.md`): custom primary agent, read + bash access, edit scoped to `report.md` only. Cannot corrupt the workspace.
