package ini

import (
	"os"
	"path/filepath"
	"strings"
)

// Update rewrites values in an ini file while preserving the user's layout:
// comments, blank lines, unknown keys and key ordering all survive. Only the
// supplied section/key pairs are rewritten in place; anything new is appended
// under its section header (creating the header if missing).
//
// Inline comments after a value are kept, since the reader strips them and a
// rewritten line would otherwise silently eat the annotation.
func Update(path string, initialSection string, sections map[string]map[string]string) error {
	if len(sections) == 0 {
		return nil
	}

	raw, err := readRawLines(path)
	if err != nil {
		raw = nil
	}

	out, leftovers := applyUpdates(raw, initialSection, sections)

	// Append remaining new keys grouped by section.
	out = appendMissing(out, leftovers)

	return atomicWrite(path, strings.Join(out, "\n")+"\n")
}

func readRawLines(path string) ([]string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Trim trailing whitespace only; keep interior blank lines and comments.
	lines := strings.Split(strings.TrimRight(string(bytes), "\n"), "\n")
	return lines, nil
}

func applyUpdates(raw []string, initialSection string, sections map[string]map[string]string) ([]string, map[string]map[string]string) {
	pending := make(map[string]map[string]string, len(sections))
	for name, keys := range sections {
		if len(keys) == 0 {
			continue
		}
		copy := make(map[string]string, len(keys))
		for k, v := range keys {
			copy[k] = v
		}
		pending[name] = copy
	}

	seen := make(map[string]bool)
	current := initialSection
	seenSections := make(map[string]bool)
	if _, ok := pending[current]; ok {
		seenSections[current] = true
	}

	out := make([]string, 0, len(raw)+16)

	for _, line := range raw {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			current = trimmed[1 : len(trimmed)-1]
			seenSections[current] = true
			out = append(out, line)
			continue
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			out = append(out, line)
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			out = append(out, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		if keys, ok := pending[current]; ok {
			if value, ok := keys[key]; ok && !seen[current+"/"+key] {
				seen[current+"/"+key] = true
				comment := inlineComment(line)
				line = key + " = " + value
				if comment != "" {
					line += "   " + comment
				}
				delete(keys, key)
				if len(keys) == 0 {
					delete(pending, current)
				}
				out = append(out, line)
				continue
			}
		}

		out = append(out, line)
	}

	// Sections that were never present in the file must not be created if all
	// their keys were consumed elsewhere (defensive).
	return out, pending
}

func inlineComment(line string) string {
	idx := strings.Index(line, "#")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[idx:])
}

func appendMissing(out []string, pending map[string]map[string]string) []string {
	// Keep deterministic order: known sections first, then sorted leftovers.
	names := make([]string, 0, len(pending))
	for name := range pending {
		names = append(names, name)
	}
	sortStrings(names)

	for _, name := range names {
		keys := pending[name]
		if len(keys) == 0 {
			continue
		}

		// Find where the section header already lives, else append.
		if !sectionExists(out, name) {
			out = append(out, "", "["+name+"]")
		}

		keyNames := make([]string, 0, len(keys))
		for k := range keys {
			keyNames = append(keyNames, k)
		}
		sortStrings(keyNames)

		// Insert after the section header, before the next section.
		headerIdx := sectionIndex(out, name)
		insertAt := headerIdx + 1
		for i := insertAt; i < len(out); i++ {
			trimmed := strings.TrimSpace(out[i])
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				break
			}
			insertAt = i + 1
		}

		block := make([]string, 0, len(keyNames))
		for _, k := range keyNames {
			block = append(block, k+" = "+keys[k])
		}
		if insertAt >= len(out) {
			out = append(out, block...)
		} else {
			after := append([]string{""}, out[insertAt:]...)
			out = append(out[:insertAt], append(block, after...)...)
		}
	}

	return out
}

func sectionExists(lines []string, name string) bool {
	return sectionIndex(lines, name) >= 0
}

func sectionIndex(lines []string, name string) int {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "["+name+"]" {
			return i
		}
	}
	return -1
}

func sortStrings(list []string) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j] < list[j-1]; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func atomicWrite(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".hypr-dock-ini-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}

	return nil
}
