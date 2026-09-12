package configedit

import (
	"bytes"
	"text/template"

	"script-manager/internal/action"
	"script-manager/internal/render"
	"script-manager/internal/secret"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var previewMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

func PreviewItem(item ItemDTO, envFields []FieldDTO, displays []DisplayDTO, displayName string, configPath string) (result PreviewDTO) {
	defer func() { result.MissingFields = nonNil(result.MissingFields) }()

	itemMap, err := FromItemDTO(item)
	if err != nil {
		return PreviewDTO{Error: err.Error()}
	}
	env, err := FieldsToMap(envFields)
	if err != nil {
		return PreviewDTO{Error: err.Error()}
	}
	merged := secret.Redact(action.Merge(env, itemMap))

	d := findDisplayDTO(displays, displayName)

	listLabel, err := action.Expand(d.List, merged)
	if err != nil {
		listLabel = d.List
	}

	funcMap := template.FuncMap{"mask": render.MaskFunc}
	tmpl, err := template.New("preview").Funcs(funcMap).Parse(d.Details)
	if err != nil {
		return PreviewDTO{ListLabel: listLabel, Error: "details template error: " + err.Error()}
	}
	data, missing := render.FillMissingFields(tmpl, merged)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return PreviewDTO{ListLabel: listLabel, Error: "details template error: " + err.Error()}
	}

	expanded := render.ExpandAllEnv(buf.String(), merged)
	expanded = render.ExpandConfigFile(expanded, configPath)
	displayMd, _, _ := render.ProcessMaskSpans(expanded)

	var htmlBuf bytes.Buffer
	if err := previewMarkdown.Convert([]byte(displayMd), &htmlBuf); err != nil {
		return PreviewDTO{ListLabel: listLabel, MissingFields: missing, DetailsHTML: "<pre>" + displayMd + "</pre>"}
	}
	return PreviewDTO{ListLabel: listLabel, DetailsHTML: htmlBuf.String(), MissingFields: missing}
}

func findDisplayDTO(displays []DisplayDTO, name string) DisplayDTO {
	if len(displays) == 0 {
		return DisplayDTO{}
	}
	if name != "" {
		for _, d := range displays {
			if d.Name == name {
				return d
			}
		}
	}
	return displays[0]
}

func PreviewAction(item ItemDTO, envFields []FieldDTO, act ActionDTO) ActionPreviewDTO {
	itemMap, err := FromItemDTO(item)
	if err != nil {
		return ActionPreviewDTO{Error: err.Error()}
	}
	env, err := FieldsToMap(envFields)
	if err != nil {
		return ActionPreviewDTO{Error: err.Error()}
	}
	merged := secret.Redact(action.Merge(env, itemMap))
	return ActionPreviewDTO{
		Description: action.Preview(act.Description, merged),
		Cmd:         action.Preview(act.Cmd, merged),
		Script:      action.Preview(act.Script, merged),
		Language:    action.Language("", action.Preview(act.Script, merged)),
	}
}
