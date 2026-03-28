package pipeline

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// ExprContext provides the variables available for expression evaluation.
type ExprContext struct {
	Env    map[string]string // env.* variables
	Git    GitContext        // git.* variables
	Matrix map[string]string // matrix.* variables
	Vars   map[string]string // arbitrary extra variables
}

// GitContext provides git-related variables for expression evaluation.
type GitContext struct {
	Branch string
	SHA    string
	Tag    string
}

// ExpressionError represents an error during expression evaluation.
type ExpressionError struct {
	Expr    string
	Message string
}

func (e *ExpressionError) Error() string {
	return fmt.Sprintf("expression error in %q: %s", e.Expr, e.Message)
}

// Interpolate replaces all ${{ ... }} expressions in a string with their
// evaluated values using the provided context.
func Interpolate(s string, ctx *ExprContext) (string, error) {
	// Regex to find ${{ ... }} patterns — non-greedy to handle nested braces.
	re := regexp.MustCompile(`\$\{\{\s*(.*?)\s*\}\}`)

	var lastErr error
	result := re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the expression inside ${{ ... }}
		inner := re.FindStringSubmatch(match)
		if len(inner) < 2 {
			return match
		}
		expr := strings.TrimSpace(inner[1])

		val, err := EvalExpression(expr, ctx)
		if err != nil {
			lastErr = err
			return match
		}
		return val
	})

	return result, lastErr
}

// EvalExpression evaluates a single expression string against the context.
// It handles variable lookups, comparisons, boolean operators, and built-in
// functions. Returns the string result of the expression.
func EvalExpression(expr string, ctx *ExprContext) (string, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", nil
	}

	// Try as a string literal first
	if val, ok := parseStringLiteral(expr); ok {
		return val, nil
	}

	// Try as a function call
	if val, ok := evalFunction(expr, ctx); ok {
		return val, nil
	}

	// Try as a variable lookup — prioritize returning the raw value
	// over boolean coercion for interpolation contexts.
	if val, ok := lookupVar(expr, ctx); ok {
		return val, nil
	}

	// Try as a boolean/comparison expression (for conditions)
	if containsOperator(expr) {
		boolResult, err := evalBoolExpr(expr, ctx)
		if err == nil {
			if boolResult {
				return "true", nil
			}
			return "false", nil
		}
	}

	return "", &ExpressionError{Expr: expr, Message: "cannot evaluate expression"}
}

// containsOperator returns true if the expression contains a comparison or
// boolean operator, indicating it should be evaluated as a boolean expression.
func containsOperator(expr string) bool {
	return strings.Contains(expr, "==") ||
		strings.Contains(expr, "!=") ||
		strings.Contains(expr, "=~") ||
		strings.Contains(expr, "&&") ||
		strings.Contains(expr, "||") ||
		strings.HasPrefix(strings.TrimSpace(expr), "!") ||
		expr == "true" || expr == "false"
}

// EvalBool evaluates an expression as a boolean for use in `when` and `if`
// conditions. Returns true/false.
func EvalBool(expr string, ctx *ExprContext) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	}

	// Strip ${{ ... }} wrapper if present
	if strings.HasPrefix(expr, "${{") && strings.HasSuffix(expr, "}}") {
		expr = strings.TrimSpace(expr[3 : len(expr)-2])
	}

	return evalBoolExpr(expr, ctx)
}

// evalBoolExpr evaluates a boolean expression. Supports:
//   - || (OR)
//   - && (AND)
//   - ! (NOT)
//   - == (equal)
//   - != (not equal)
//   - =~ (regex match)
//   - variable references that evaluate to "true"/"false"
func evalBoolExpr(expr string, ctx *ExprContext) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Handle || (lowest precedence) — split on || not inside quotes
	if parts := splitBoolOp(expr, "||"); len(parts) > 1 {
		for _, part := range parts {
			val, err := evalBoolExpr(part, ctx)
			if err != nil {
				return false, err
			}
			if val {
				return true, nil
			}
		}
		return false, nil
	}

	// Handle && (next precedence)
	if parts := splitBoolOp(expr, "&&"); len(parts) > 1 {
		for _, part := range parts {
			val, err := evalBoolExpr(part, ctx)
			if err != nil {
				return false, err
			}
			if !val {
				return false, nil
			}
		}
		return true, nil
	}

	// Handle ! (NOT) prefix
	if strings.HasPrefix(expr, "!") {
		inner := strings.TrimSpace(expr[1:])
		val, err := evalBoolExpr(inner, ctx)
		if err != nil {
			return false, err
		}
		return !val, nil
	}

	// Handle parenthesized expression
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		inner := expr[1 : len(expr)-1]
		return evalBoolExpr(inner, ctx)
	}

	// Handle comparison operators
	if result, ok := evalComparison(expr, ctx); ok {
		return result, nil
	}

	// Handle function calls that return boolean-like values
	if val, ok := evalFunction(expr, ctx); ok {
		return val == "true", nil
	}

	// Handle bare variable reference as truthy check
	if val, ok := lookupVar(expr, ctx); ok {
		return val != "" && val != "false" && val != "0", nil
	}

	// Handle literal booleans
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}

	return false, &ExpressionError{Expr: expr, Message: "cannot evaluate as boolean"}
}

// evalComparison handles ==, !=, and =~ operators.
func evalComparison(expr string, ctx *ExprContext) (bool, bool) {
	// Try =~ (regex match) first since it contains =
	if idx := strings.Index(expr, "=~"); idx > 0 {
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+2:])

		leftVal := resolveValue(left, ctx)
		rightVal := resolveValue(right, ctx)

		// Strip regex delimiters if present
		rightVal = strings.TrimPrefix(rightVal, "/")
		rightVal = strings.TrimSuffix(rightVal, "/")

		matched, err := regexp.MatchString(rightVal, leftVal)
		if err != nil {
			return false, true
		}
		return matched, true
	}

	// Try != before ==
	if idx := strings.Index(expr, "!="); idx > 0 {
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+2:])

		leftVal := resolveValue(left, ctx)
		rightVal := resolveValue(right, ctx)
		return leftVal != rightVal, true
	}

	// Try ==
	if idx := strings.Index(expr, "=="); idx > 0 {
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+2:])

		leftVal := resolveValue(left, ctx)
		rightVal := resolveValue(right, ctx)
		return leftVal == rightVal, true
	}

	return false, false
}

// resolveValue resolves a value token — either a variable reference, string
// literal, or literal value.
func resolveValue(token string, ctx *ExprContext) string {
	token = strings.TrimSpace(token)

	// String literal
	if val, ok := parseStringLiteral(token); ok {
		return val
	}

	// Variable reference
	if val, ok := lookupVar(token, ctx); ok {
		return val
	}

	// Function call
	if val, ok := evalFunction(token, ctx); ok {
		return val
	}

	return token
}

// lookupVar resolves a dotted variable reference against the context.
// Supports env.*, git.*, matrix.*, and secrets.*.
func lookupVar(name string, ctx *ExprContext) (string, bool) {
	if ctx == nil {
		return "", false
	}

	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 {
		// Check vars map for simple names
		if ctx.Vars != nil {
			if val, ok := ctx.Vars[name]; ok {
				return val, true
			}
		}
		return "", false
	}

	namespace := parts[0]
	key := parts[1]

	switch namespace {
	case "env":
		if ctx.Env != nil {
			if val, ok := ctx.Env[key]; ok {
				return val, true
			}
		}
	case "git":
		switch key {
		case "branch":
			return ctx.Git.Branch, true
		case "sha":
			return ctx.Git.SHA, true
		case "tag":
			return ctx.Git.Tag, true
		}
	case "matrix":
		if ctx.Matrix != nil {
			if val, ok := ctx.Matrix[key]; ok {
				return val, true
			}
		}
	case "secrets":
		// Secrets are resolved at runtime; return a placeholder.
		return "***", true
	}

	return "", false
}

// evalFunction evaluates built-in function calls:
//   - hash(value) — SHA256 hash of the value (first 16 hex chars)
//   - contains(haystack, needle) — checks if haystack contains needle
func evalFunction(expr string, ctx *ExprContext) (string, bool) {
	// Parse function name and arguments
	parenIdx := strings.Index(expr, "(")
	if parenIdx < 0 || !strings.HasSuffix(expr, ")") {
		return "", false
	}

	funcName := strings.TrimSpace(expr[:parenIdx])
	argsStr := expr[parenIdx+1 : len(expr)-1]

	switch funcName {
	case "hash":
		arg := strings.TrimSpace(argsStr)
		val := resolveValue(arg, ctx)
		h := sha256.Sum256([]byte(val))
		return fmt.Sprintf("%x", h[:8]), true

	case "contains":
		args := splitFuncArgs(argsStr)
		if len(args) != 2 {
			return "", false
		}
		haystack := resolveValue(args[0], ctx)
		needle := resolveValue(args[1], ctx)
		if strings.Contains(haystack, needle) {
			return "true", true
		}
		return "false", true

	default:
		return "", false
	}
}

// parseStringLiteral extracts a string value from single or double quotes.
func parseStringLiteral(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') ||
			(s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1], true
		}
	}
	return "", false
}

// splitBoolOp splits an expression on a boolean operator (&&, ||) while
// respecting parentheses and quoted strings.
func splitBoolOp(expr string, op string) []string {
	var parts []string
	depth := 0
	inSingle := false
	inDouble := false
	start := 0

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if inSingle || inDouble {
			continue
		}

		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
		}

		if depth == 0 && i+len(op) <= len(expr) && expr[i:i+len(op)] == op {
			parts = append(parts, expr[start:i])
			start = i + len(op)
			i += len(op) - 1
		}
	}

	parts = append(parts, expr[start:])

	// If we only got one part, the operator was not found at the top level.
	if len(parts) == 1 {
		return parts
	}

	// Trim whitespace
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	return parts
}

// splitFuncArgs splits function arguments on commas while respecting quotes
// and parentheses.
func splitFuncArgs(s string) []string {
	var args []string
	depth := 0
	inSingle := false
	inDouble := false
	start := 0

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
		} else if ch == '"' && !inSingle {
			inDouble = !inDouble
		} else if !inSingle && !inDouble {
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
			} else if ch == ',' && depth == 0 {
				args = append(args, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}

	args = append(args, strings.TrimSpace(s[start:]))
	return args
}

// InterpolateJob interpolates all ${{ ... }} expressions in a JobSpec.
func InterpolateJob(job *JobSpec, ctx *ExprContext) error {
	var err error

	// Interpolate image
	if job.Image != "" {
		job.Image, err = Interpolate(job.Image, ctx)
		if err != nil {
			return err
		}
	}

	// Interpolate env values
	for k, v := range job.Env {
		job.Env[k], err = Interpolate(v, ctx)
		if err != nil {
			return err
		}
	}

	// Interpolate cache keys
	for i := range job.Cache {
		job.Cache[i].Key, err = Interpolate(job.Cache[i].Key, ctx)
		if err != nil {
			return err
		}
	}

	// Interpolate steps
	for i := range job.Steps {
		if err := interpolateStep(&job.Steps[i], ctx); err != nil {
			return err
		}
	}

	return nil
}

// interpolateStep interpolates all ${{ ... }} expressions in a StepSpec.
func interpolateStep(step *StepSpec, ctx *ExprContext) error {
	var err error

	if step.Uses != "" {
		step.Uses, err = Interpolate(step.Uses, ctx)
		if err != nil {
			return err
		}
	}

	if step.Run != "" {
		step.Run, err = Interpolate(step.Run, ctx)
		if err != nil {
			return err
		}
	}

	for k, v := range step.With {
		step.With[k], err = Interpolate(v, ctx)
		if err != nil {
			return err
		}
	}

	for k, v := range step.Env {
		step.Env[k], err = Interpolate(v, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

