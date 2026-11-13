package notes

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// NoteType identifies an optional Zettelkasten note type.
type NoteType string

const (
	Fleeting   NoteType = "fleeting"
	Literature NoteType = "literature"
	Permanent  NoteType = "permanent"
)

// readMetadata recognizes a YAML mapping between standalone opening and closing
// delimiters. It never changes the original note; malformed metadata remains
// part of the content so it can still be viewed and edited.
type metadata struct {
	Type    NoteType
	Tags    []string
	Sources []string
}

func readMetadata(body string) (metadata, string, string) {
	opening, remaining, ok := strings.Cut(body, "\n")
	if !ok || strings.TrimSuffix(opening, "\r") != "---" {
		return metadata{}, body, ""
	}
	var front strings.Builder
	for len(remaining) > 0 {
		line, rest, hasNewline := strings.Cut(remaining, "\n")
		if strings.TrimSuffix(line, "\r") == "---" {
			var meta struct {
				Type    string    `yaml:"type"`
				Tags    yaml.Node `yaml:"tags"`
				Sources yaml.Node `yaml:"sources"`
			}
			if err := yaml.Unmarshal([]byte(front.String()), &meta); err != nil {
				return metadata{}, body, fmt.Sprintf("invalid frontmatter: %v", err)
			}
			content := strings.TrimLeft(rest, "\r\n")
			var result metadata
			var warnings []string
			switch NoteType(meta.Type) {
			case "":
			case Fleeting, Literature, Permanent:
				result.Type = NoteType(meta.Type)
			default:
				warnings = append(warnings, fmt.Sprintf("unknown note type %q", meta.Type))
			}
			if meta.Tags.Kind != 0 && meta.Tags.Tag != "!!null" {
				if meta.Tags.Kind != yaml.SequenceNode {
					warnings = append(warnings, "tags must be a list of non-empty strings")
				} else {
					seen := map[string]bool{}
					for _, tag := range meta.Tags.Content {
						if tag.Kind != yaml.ScalarNode || tag.Tag != "!!str" || strings.TrimSpace(tag.Value) == "" {
							warnings = append(warnings, "tags must be non-empty strings")
							continue
						}
						value := strings.TrimSpace(tag.Value)
						if !seen[value] {
							result.Tags = append(result.Tags, value)
							seen[value] = true
						}
					}
				}
			}
			if meta.Sources.Kind != 0 && meta.Sources.Tag != "!!null" {
				if meta.Sources.Kind != yaml.SequenceNode {
					warnings = append(warnings, "sources must be a list of non-empty strings")
				} else {
					for _, source := range meta.Sources.Content {
						if source.Kind != yaml.ScalarNode || source.Tag != "!!str" || strings.TrimSpace(source.Value) == "" {
							warnings = append(warnings, "sources must be non-empty strings")
							continue
						}
						result.Sources = append(result.Sources, strings.TrimSpace(source.Value))
					}
				}
			}
			return result, content, strings.Join(warnings, "; ")
		}
		front.WriteString(line)
		front.WriteByte('\n')
		if !hasNewline {
			break
		}
		remaining = rest
	}
	return metadata{}, body, ""
}
