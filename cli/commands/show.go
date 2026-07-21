package commands

import "drift/cli/output"

// ShowCommand implements `drift show <marker|spec>`.
type ShowCommand struct{}

// D! id=cshow range-start
func (c ShowCommand) Run(ctx Context) (output.Result, int) {
	if len(ctx.Args) < 2 {
		return output.ErrorResult{
			Command: "show",
			Message: "usage: drift show <marker|spec> [--no-content]\n\nExample: drift show cval\n         drift show core.validate\n         drift show core.validate --no-content",
			Exit:    1,
		}, 1
	}
	state, err := ctx.Orch.Todo(ctx.Sess)
	if err != nil {
		return output.ErrorResult{Command: "show", Message: err.Error(), Exit: 1}, 1
	}
	result, err := output.BuildShowResult(state, ctx.Dir, ctx.Args[1])
	if err != nil {
		return output.ErrorResult{Command: "show", Message: err.Error(), Exit: 1}, 1
	}
	if hasFlag(ctx.Args, "--no-content") {
		for i := range result.Nodes {
			result.Nodes[i].Content = ""
		}
	}
	code := 0
	if !output.EntityExists(state, ctx.Args[1]) {
		code = 1
	}
	return result, code
}

// hasFlag reports whether flag appears in args (1..n).
func hasFlag(args []string, flag string) bool {
	for _, a := range args[1:] {
		if a == flag {
			return true
		}
	}
	return false
}

// D! id=cshow range-end

func (c ShowCommand) Meta() Meta {
	return Meta{
		Name:  "show",
		Short: "Show citation closure of a spec or marker",
		Usage: "Usage: drift show <marker|spec> [--no-content]\n\nShow the full citation closure reachable from the seed: every spec\ntransitively connected (ancestors + descendants), every marker linked\nto any reached spec, and every edge among them. Diamond/forking cases\npreserved in the edges list.\n\nFlags:\n  --no-content  Omit Content from each node. Useful for fetching the\n                graph overview cheaply; content can be fetched per-spec\n                via a second call.\n\nExamples:\n  drift show cval\n  drift show core.validate\n  drift show core.validate --no-content",
		Flags: []string{"--no-content"},
	}
}