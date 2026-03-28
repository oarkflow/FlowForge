package pipeline

import (
	"testing"
)

func TestInterpolate_EnvVar(t *testing.T) {
	ctx := &ExprContext{
		Env: map[string]string{"GO_VERSION": "1.22"},
	}
	result, err := Interpolate("golang:${{ env.GO_VERSION }}-alpine", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "golang:1.22-alpine" {
		t.Errorf("got %q, want %q", result, "golang:1.22-alpine")
	}
}

func TestInterpolate_GitBranch(t *testing.T) {
	ctx := &ExprContext{
		Git: GitContext{Branch: "main", SHA: "abc123"},
	}
	result, err := Interpolate("branch=${{ git.branch }}, sha=${{ git.sha }}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "branch=main, sha=abc123" {
		t.Errorf("got %q", result)
	}
}

func TestInterpolate_MatrixVar(t *testing.T) {
	ctx := &ExprContext{
		Matrix: map[string]string{"go_version": "1.21"},
	}
	result, err := Interpolate("golang:${{ matrix.go_version }}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "golang:1.21" {
		t.Errorf("got %q", result)
	}
}

func TestInterpolate_Secrets(t *testing.T) {
	ctx := &ExprContext{}
	result, err := Interpolate("${{ secrets.API_KEY }}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "***" {
		t.Errorf("got %q, want %q", result, "***")
	}
}

func TestInterpolate_NoExpressions(t *testing.T) {
	ctx := &ExprContext{}
	input := "plain text with no expressions"
	result, err := Interpolate(input, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != input {
		t.Errorf("got %q, want %q", result, input)
	}
}

func TestInterpolate_HashFunction(t *testing.T) {
	ctx := &ExprContext{}
	result, err := Interpolate("${{ hash('go.sum') }}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 16 {
		t.Errorf("hash result length = %d, want 16", len(result))
	}
}

func TestInterpolate_ContainsFunction(t *testing.T) {
	ctx := &ExprContext{
		Env: map[string]string{"TAGS": "latest,v1.0"},
	}
	result, err := Interpolate("${{ contains(env.TAGS, 'latest') }}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "true" {
		t.Errorf("got %q, want %q", result, "true")
	}
}

func TestEvalBool_Equality(t *testing.T) {
	ctx := &ExprContext{
		Git: GitContext{Branch: "main"},
	}
	tests := []struct {
		expr string
		want bool
	}{
		{`git.branch == 'main'`, true},
		{`git.branch == 'develop'`, false},
		{`git.branch != 'develop'`, true},
		{`git.branch != 'main'`, false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalBool(tt.expr, ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("EvalBool(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEvalBool_RegexMatch(t *testing.T) {
	ctx := &ExprContext{
		Git: GitContext{Tag: "v1.2.3"},
	}
	got, err := EvalBool(`git.tag =~ /^v\d+/`, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("regex match should be true")
	}
}

func TestEvalBool_BooleanOperators(t *testing.T) {
	ctx := &ExprContext{
		Git: GitContext{Branch: "main"},
		Env: map[string]string{"DEPLOY": "true"},
	}
	tests := []struct {
		expr string
		want bool
	}{
		{`git.branch == 'main' && env.DEPLOY == 'true'`, true},
		{`git.branch == 'main' && env.DEPLOY == 'false'`, false},
		{`git.branch == 'develop' || env.DEPLOY == 'true'`, true},
		{`git.branch == 'develop' || env.DEPLOY == 'false'`, false},
		{`!false`, true},
		{`!true`, false},
		{`true`, true},
		{`false`, false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalBool(tt.expr, ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("EvalBool(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEvalBool_EmptyExpression(t *testing.T) {
	got, err := EvalBool("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("empty expression should evaluate to true")
	}
}

func TestEvalBool_StripDollarBraces(t *testing.T) {
	ctx := &ExprContext{Git: GitContext{Branch: "main"}}
	got, err := EvalBool("${{ git.branch == 'main' }}", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("should strip ${{ }} wrapper and evaluate")
	}
}

func TestEvalBool_TruthyVariable(t *testing.T) {
	ctx := &ExprContext{
		Vars: map[string]string{"was_failing": "true"},
	}
	got, err := EvalBool("was_failing", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("truthy variable should evaluate to true")
	}
}

func TestEvalBool_FalsyVariable(t *testing.T) {
	ctx := &ExprContext{
		Vars: map[string]string{"flag": "false"},
	}
	got, err := EvalBool("flag", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("'false' variable should evaluate to false")
	}
}

func TestEvalBool_Parentheses(t *testing.T) {
	ctx := &ExprContext{}
	got, err := EvalBool("(true)", ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("(true) should be true")
	}
}

func TestEvalExpression_StringLiteral(t *testing.T) {
	val, err := EvalExpression(`'hello'`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Errorf("got %q, want %q", val, "hello")
	}
}

func TestEvalExpression_UnknownExpression(t *testing.T) {
	_, err := EvalExpression("unknown_thing", &ExprContext{})
	if err == nil {
		t.Error("should return error for unknown expression")
	}
}

func TestLookupVar_NilContext(t *testing.T) {
	_, ok := lookupVar("env.KEY", nil)
	if ok {
		t.Error("should return false for nil context")
	}
}

func TestInterpolateJob(t *testing.T) {
	job := &JobSpec{
		Image: "golang:${{ env.GO_VERSION }}",
		Env:   map[string]string{"APP": "${{ env.APP_NAME }}"},
		Cache: CacheList{{Key: "mod-${{ hash('go.sum') }}"}},
		Steps: []StepSpec{
			{Run: "echo ${{ env.MSG }}", Env: map[string]string{"V": "${{ env.VAL }}"}},
		},
	}
	ctx := &ExprContext{
		Env: map[string]string{
			"GO_VERSION": "1.22",
			"APP_NAME":   "myapp",
			"MSG":        "hello",
			"VAL":        "42",
		},
	}
	if err := InterpolateJob(job, ctx); err != nil {
		t.Fatal(err)
	}
	if job.Image != "golang:1.22" {
		t.Errorf("Image = %q", job.Image)
	}
	if job.Env["APP"] != "myapp" {
		t.Errorf("Env[APP] = %q", job.Env["APP"])
	}
	if job.Steps[0].Env["V"] != "42" {
		t.Errorf("Step.Env[V] = %q", job.Steps[0].Env["V"])
	}
}
