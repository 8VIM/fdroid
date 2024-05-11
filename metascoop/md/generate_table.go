package md

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"metascoop/apps"
)

const (
	tableStart = "<!-- This table is auto-generated. Do not edit -->"

	tableEnd = "<!-- end apps table -->"

	tableTmpl = `
| Icon | Name | Description | Version |
| --- | --- | --- | --- |{{range .Apps}}
| 
{{- $name := .name -}}
{{- $sourceCode := .sourceCode -}}
{{- $packageName := .packageName -}}
{{- with (index .localized "en-US").icon -}}
<a href="{{$sourceCode}}"><img src="fdroid/repo/{{$packageName}}/en-US/{{.}}" alt="{{$name}} icon" width="36px" height="36px"></a>
{{- end -}} | [**{{$name}}**]({{$sourceCode}}) | {{.summary}} | {{.suggestedVersionName}} ({{.suggestedVersionCode}}) |{{end}}
` + tableEnd
)

var tmpl = template.Must(template.New("").Parse(tableTmpl))

func RegenerateReadme(repoDir string) (err error) {
	readmePath := filepath.Join(filepath.Dir(filepath.Dir(repoDir)), "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return
	}

	fdroidIndexFilePath := filepath.Join(repoDir, "index-v1.json")
	var index *apps.RepoIndex
	index, err = apps.ReadIndex(fdroidIndexFilePath)
	if err != nil {
		err = fmt.Errorf("reading f-droid repo index: %s\n::endgroup::\n", err.Error())
		return
	}

	var tableStartIndex = bytes.Index(content, []byte(tableStart))
	if tableStartIndex < 0 {
		return fmt.Errorf("cannot find table start in %q", readmePath)
	}

	var tableEndIndex = bytes.Index(content, []byte(tableEnd))
	if tableEndIndex < 0 {
		return fmt.Errorf("cannot find table end in %q", readmePath)
	}

	var table bytes.Buffer

	table.WriteString(tableStart)

	err = tmpl.Execute(&table, index)
	if err != nil {
		return err
	}

	newContent := []byte{}

	newContent = append(newContent, content[:tableStartIndex]...)
	newContent = append(newContent, table.Bytes()...)
	newContent = append(newContent, content[tableEndIndex:]...)

	return os.WriteFile(readmePath, newContent, os.ModePerm)
}
