// Package lint validates the layout and metadata of an OEFP checkout.
package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	frontmatter  = regexp.MustCompile(`(?s)/-!(.*?)-/`)
	theoremID    = regexp.MustCompile(`^T([0-9]{8})$`)
	definitionID = regexp.MustCompile(`^D([0-9]{8})$`)
)

var allowedFields = map[string]bool{
	"id": true, "title": true, "aliases": true, "description": true,
	"tags": true, "status": true,
}

// Result describes one lint run. Errors are returned in stable path and field
// order so command output is suitable for CI logs.
type Result struct {
	Files  int
	Errors []string
}

// Check validates every Lean entry beneath root/OEFP.
func Check(root string) Result {
	entries := filepath.Join(root, "OEFP")
	if info, err := os.Stat(entries); err != nil || !info.IsDir() {
		return Result{Errors: []string{fmt.Sprintf("%s: entry directory not found", entries)}}
	}

	var files []string
	var walkErrors []string
	_ = filepath.WalkDir(entries, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			walkErrors = append(walkErrors, fmt.Sprintf("%s: cannot inspect path: %v", path, err))
			return nil
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".lean") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)

	result := Result{Files: len(files), Errors: walkErrors}
	for _, path := range files {
		relative, _ := filepath.Rel(root, path)
		entryID := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		expected, ok := expectedPath(entryID)
		if !ok {
			result.Errors = append(result.Errors,
				fmt.Sprintf("%s: filename must be T or D followed by 8 digits", relative))
			continue
		}
		if filepath.Clean(relative) != expected {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: entry belongs at %s", relative, expected))
		}
		result.Errors = append(result.Errors, validateFrontmatter(path, relative, entryID)...)
	}
	return result
}

func expectedPath(id string) (string, bool) {
	if match := theoremID.FindStringSubmatch(id); match != nil {
		digits := match[1]
		return filepath.Join("OEFP", "T"+digits[:2], "T"+digits[:4], "T"+digits[:6], id+".lean"), true
	}
	if match := definitionID.FindStringSubmatch(id); match != nil {
		digits := match[1]
		return filepath.Join("OEFP", "D"+digits[:2], "D"+digits[:4], "D"+digits[:6], id+".lean"), true
	}
	return "", false
}

func validateFrontmatter(path, relative, entryID string) []string {
	source, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: cannot read file: %v", relative, err)}
	}
	block := frontmatter.FindSubmatch(source)
	if block == nil {
		return []string{fmt.Sprintf("%s: missing Lean module doc comment with YAML front matter (/-! ... -/)", relative)}
	}

	var document yaml.Node
	if err := yaml.Unmarshal(block[1], &document); err != nil {
		return []string{fmt.Sprintf("%s: invalid YAML front matter: %v", relative, err)}
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return []string{fmt.Sprintf("%s: front matter must be a YAML mapping", relative)}
	}

	fields := map[string]*yaml.Node{}
	var errors []string
	mapping := document.Content[0]
	for i := 0; i < len(mapping.Content); i += 2 {
		name, value := mapping.Content[i].Value, mapping.Content[i+1]
		if !allowedFields[name] {
			errors = append(errors, fmt.Sprintf("%s: invalid front matter: unknown field %q", relative, name))
			continue
		}
		fields[name] = value
	}

	for _, name := range []string{"id", "title", "description", "tags"} {
		if fields[name] == nil {
			errors = append(errors, fmt.Sprintf("%s: invalid front matter: missing required field %q", relative, name))
		}
	}
	for _, name := range []string{"id", "title", "description"} {
		if node := fields[name]; node != nil && (node.Kind != yaml.ScalarNode || node.Tag != "!!str" || node.Value == "") {
			errors = append(errors, fmt.Sprintf("%s: invalid front matter at %s: must be a non-empty string", relative, name))
		}
	}
	if node := fields["id"]; node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		if !theoremID.MatchString(node.Value) && !definitionID.MatchString(node.Value) {
			errors = append(errors, fmt.Sprintf("%s: invalid front matter at id: must be a theorem or definition ID", relative))
		}
		if node.Value != entryID {
			errors = append(errors, fmt.Sprintf("%s: front matter id %q does not match filename id %s", relative, node.Value, entryID))
		}
	}
	if node := fields["aliases"]; node != nil {
		errors = append(errors, validateStringList(relative, "aliases", node, false)...)
	}
	if node := fields["tags"]; node != nil {
		errors = append(errors, validateStringList(relative, "tags", node, true)...)
	}
	status := fields["status"]
	metadataID := fields["id"]
	if metadataID != nil && metadataID.Kind == yaml.ScalarNode && strings.HasPrefix(metadataID.Value, "T") && status == nil {
		errors = append(errors, fmt.Sprintf("%s: invalid front matter: missing required field %q", relative, "status"))
	}
	if status != nil && (status.Kind != yaml.ScalarNode || status.Tag != "!!str" || (status.Value != "proved" && status.Value != "open")) {
		errors = append(errors, fmt.Sprintf("%s: invalid front matter at status: must be %q or %q", relative, "proved", "open"))
	}
	return errors
}

func validateStringList(relative, field string, node *yaml.Node, nonempty bool) []string {
	if node.Kind != yaml.SequenceNode {
		return []string{fmt.Sprintf("%s: invalid front matter at %s: must be a list of non-empty strings", relative, field)}
	}
	if nonempty && len(node.Content) == 0 {
		return []string{fmt.Sprintf("%s: invalid front matter at %s: must contain at least one item", relative, field)}
	}
	for _, item := range node.Content {
		if item.Kind != yaml.ScalarNode || item.Tag != "!!str" || item.Value == "" {
			return []string{fmt.Sprintf("%s: invalid front matter at %s: must be a list of non-empty strings", relative, field)}
		}
	}
	return nil
}
