package render

import (
	"maps"
	"sort"
	"strings"
	"text/template"
	"text/template/parse"
)

const missingFieldValue = "<nil>"

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

func MissingFieldsWarning(missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	noun := "value"
	if len(missing) > 1 {
		noun = "values"
	}
	return "⚠️ *Missing " + noun + " (shown as &lt;nil&gt;): " + strings.Join(missing, ", ") + "*\n\n"
}

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
		if len(a.Ident) > 1 && a.Ident[0] == "$" {
			out[a.Ident[1]] = true
		}
	case *parse.ChainNode:
		collectArg(a.Node, out)
	case *parse.PipeNode:
		collectPipe(a, out)
	}
}
