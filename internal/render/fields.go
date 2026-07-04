package render

import (
	"maps"
	"sort"
	"strings"
	"text/template"
	"text/template/parse"
)

// missingFieldValue substitutes for fields the template references but the
// item doesn't have. A visible literal (matching what fmt prints for a nil)
// rather than "" so the gap is obvious in place and empty code spans / table
// cells don't render oddly.
const missingFieldValue = "<nil>"

// FillMissingFields returns the data to execute a details template with:
// every top-level field the template references (e.g. {{.foo}} or
// {{mask .foo}}) that is absent from item — or present but nil — is set to
// missingFieldValue in a copy of item, so execution renders a visible
// placeholder instead of failing with "invalid value; expected string". The
// second return value lists the filled field names, sorted, for surfacing a
// warning; it is nil when nothing was missing, in which case item is
// returned unmodified.
func FillMissingFields(tmpl *template.Template, item map[string]any) (map[string]any, []string) {
	if tmpl == nil || tmpl.Tree == nil || tmpl.Tree.Root == nil || item == nil {
		return item, nil
	}
	fields := make(map[string]bool)
	collectFields(tmpl.Tree.Root, fields)

	var missing []string
	for f := range fields {
		if v, ok := item[f]; !ok || v == nil {
			missing = append(missing, f)
		}
	}
	if len(missing) == 0 {
		return item, nil
	}
	sort.Strings(missing)

	filled := make(map[string]any, len(item)+len(missing))
	maps.Copy(filled, item)
	for _, f := range missing {
		filled[f] = missingFieldValue
	}
	return filled, missing
}

// MissingFieldsWarning renders the warning block prepended to the details
// Markdown when FillMissingFields filled anything. Field names are not
// backtick-wrapped on purpose: backtick spans become copy targets, and a
// missing field has no value worth copying.
func MissingFieldsWarning(missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	noun := "value"
	if len(missing) > 1 {
		noun = "values"
	}
	// &lt;/&gt; because a literal <nil> would parse as a raw inline HTML tag
	// and vanish from the GUI's rendered output.
	return "⚠️ *Missing " + noun + " (shown as &lt;nil&gt;): " + strings.Join(missing, ", ") + "*\n\n"
}

// collectFields walks a template parse tree and records the first identifier
// of every field reference. Fields inside {{with}}/{{range}} bodies are
// resolved against the loop/with value at execution time, not the top-level
// item, so collecting them can over-report — acceptable here because details
// templates render a flat item map.
func collectFields(node parse.Node, out map[string]bool) {
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, c := range n.Nodes {
			collectFields(c, out)
		}
	case *parse.ActionNode:
		collectPipe(n.Pipe, out)
	case *parse.IfNode:
		collectBranch(&n.BranchNode, out)
	case *parse.RangeNode:
		collectBranch(&n.BranchNode, out)
	case *parse.WithNode:
		collectBranch(&n.BranchNode, out)
	case *parse.TemplateNode:
		collectPipe(n.Pipe, out)
	}
}

func collectBranch(b *parse.BranchNode, out map[string]bool) {
	collectPipe(b.Pipe, out)
	collectFields(b.List, out)
	collectFields(b.ElseList, out)
}

func collectPipe(pipe *parse.PipeNode, out map[string]bool) {
	if pipe == nil {
		return
	}
	for _, cmd := range pipe.Cmds {
		for _, arg := range cmd.Args {
			collectArg(arg, out)
		}
	}
}

func collectArg(arg parse.Node, out map[string]bool) {
	switch a := arg.(type) {
	case *parse.FieldNode:
		if len(a.Ident) > 0 {
			out[a.Ident[0]] = true
		}
	case *parse.VariableNode:
		// $.foo reaches the top-level item regardless of nesting.
		if len(a.Ident) > 1 && a.Ident[0] == "$" {
			out[a.Ident[1]] = true
		}
	case *parse.ChainNode:
		collectArg(a.Node, out)
	case *parse.PipeNode:
		collectPipe(a, out)
	}
}
