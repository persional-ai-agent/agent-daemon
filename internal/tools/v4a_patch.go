package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type V4AOperationType string

const (
	V4AOpAdd    V4AOperationType = "add"
	V4AOpUpdate V4AOperationType = "update"
	V4AOpDelete V4AOperationType = "delete"
	V4AOpMove   V4AOperationType = "move"
)

type V4AHunkLine struct {
	Prefix  byte   // ' ', '+', '-'
	Content string // without prefix
}

type V4AHunk struct {
	ContextHint string
	Lines       []V4AHunkLine
}

type V4AOperation struct {
	Op      V4AOperationType
	Path    string
	NewPath string // for move
	Hunks   []V4AHunk
	Content string // for add
}

var (
	reUpdate = regexp.MustCompile(`^\*\*\*\s*Update\s+File:\s*(.+)\s*$`)
	reAdd    = regexp.MustCompile(`^\*\*\*\s*Add\s+File:\s*(.+)\s*$`)
	reDelete = regexp.MustCompile(`^\*\*\*\s*Delete\s+File:\s*(.+)\s*$`)
	reMove   = regexp.MustCompile(`^\*\*\*\s*Move\s+File:\s*(.+?)\s*->\s*(.+)\s*$`)
	reHunk   = regexp.MustCompile(`^@@\s*(.*?)\s*@@\s*$`)
)

func ParseV4APatch(patch string) ([]V4AOperation, error) {
	lines := strings.Split(patch, "\n")
	start := 0
	end := len(lines)
	for i, line := range lines {
		if strings.Contains(line, "*** Begin Patch") || strings.Contains(line, "***Begin Patch") {
			start = i + 1
		}
		if strings.Contains(line, "*** End Patch") || strings.Contains(line, "***End Patch") {
			end = i
			break
		}
	}

	var ops []V4AOperation
	var cur *V4AOperation
	var curHunk *V4AHunk

	flush := func() {
		if cur == nil {
			return
		}
		if curHunk != nil && len(curHunk.Lines) > 0 {
			cur.Hunks = append(cur.Hunks, *curHunk)
		}
		ops = append(ops, *cur)
		cur = nil
		curHunk = nil
	}

	for i := start; i < end; i++ {
		line := lines[i]
		if m := reUpdate.FindStringSubmatch(line); m != nil {
			flush()
			cur = &V4AOperation{Op: V4AOpUpdate, Path: strings.TrimSpace(m[1])}
			curHunk = nil
			continue
		}
		if m := reAdd.FindStringSubmatch(line); m != nil {
			flush()
			cur = &V4AOperation{Op: V4AOpAdd, Path: strings.TrimSpace(m[1])}
			curHunk = &V4AHunk{}
			continue
		}
		if m := reDelete.FindStringSubmatch(line); m != nil {
			flush()
			ops = append(ops, V4AOperation{Op: V4AOpDelete, Path: strings.TrimSpace(m[1])})
			continue
		}
		if m := reMove.FindStringSubmatch(line); m != nil {
			flush()
			ops = append(ops, V4AOperation{Op: V4AOpMove, Path: strings.TrimSpace(m[1]), NewPath: strings.TrimSpace(m[2])})
			continue
		}
		if m := reHunk.FindStringSubmatch(line); m != nil {
			if cur == nil {
				continue
			}
			if curHunk != nil && len(curHunk.Lines) > 0 {
				cur.Hunks = append(cur.Hunks, *curHunk)
			}
			curHunk = &V4AHunk{ContextHint: strings.TrimSpace(m[1])}
			continue
		}

		if cur == nil {
			continue
		}
		if curHunk == nil {
			curHunk = &V4AHunk{}
		}
		if strings.HasPrefix(line, `\`) {
			continue
		}
		if line == "" {
			curHunk.Lines = append(curHunk.Lines, V4AHunkLine{Prefix: ' ', Content: ""})
			continue
		}
		switch line[0] {
		case '+', '-', ' ':
			curHunk.Lines = append(curHunk.Lines, V4AHunkLine{Prefix: line[0], Content: line[1:]})
		default:
			curHunk.Lines = append(curHunk.Lines, V4AHunkLine{Prefix: ' ', Content: line})
		}
	}
	if cur != nil {
		flush()
	}

	for _, op := range ops {
		if strings.TrimSpace(op.Path) == "" {
			return nil, fmt.Errorf("invalid patch: empty file path")
		}
		if op.Op == V4AOpUpdate && len(op.Hunks) == 0 {
			return nil, fmt.Errorf("invalid patch: UPDATE %q has no hunks", op.Path)
		}
		if op.Op == V4AOpMove && strings.TrimSpace(op.NewPath) == "" {
			return nil, fmt.Errorf("invalid patch: MOVE %q missing destination path", op.Path)
		}
	}

	return ops, nil
}

func ApplyV4AOperations(ops []V4AOperation, workdir string) (map[string]any, error) {
	changed := make([]string, 0)
	deleted := make([]string, 0)
	added := make([]string, 0)
	moved := make([]map[string]any, 0)

	for _, op := range ops {
		switch op.Op {
		case V4AOpAdd:
			target, err := resolvePathWithinWorkdir(workdir, op.Path)
			if err != nil {
				return nil, err
			}
			if err := rejectSymlinkEscape(workdir, target); err != nil {
				return nil, err
			}
			if _, err := os.Stat(target); err == nil {
				return nil, fmt.Errorf("add file already exists: %s", op.Path)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, err
			}
			content := op.Content
			if content == "" {
				var out []string
				for _, h := range op.Hunks {
					for _, ln := range h.Lines {
						if ln.Prefix == '+' {
							out = append(out, ln.Content)
						} else if ln.Prefix == ' ' && ln.Content != "" {
							// be tolerant: treat unprefixed/context lines as content for add ops
							out = append(out, ln.Content)
						}
					}
				}
				content = strings.Join(out, "\n")
			}
			if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
				return nil, err
			}
			added = append(added, target)
		case V4AOpDelete:
			target, err := resolvePathWithinWorkdir(workdir, op.Path)
			if err != nil {
				return nil, err
			}
			if err := rejectSymlinkEscape(workdir, target); err != nil {
				return nil, err
			}
			if err := rejectNonRegularFile(target); err != nil {
				return nil, err
			}
			if err := os.Remove(target); err != nil {
				return nil, err
			}
			deleted = append(deleted, target)
		case V4AOpMove:
			src, err := resolvePathWithinWorkdir(workdir, op.Path)
			if err != nil {
				return nil, err
			}
			dst, err := resolvePathWithinWorkdir(workdir, op.NewPath)
			if err != nil {
				return nil, err
			}
			if err := rejectSymlinkEscape(workdir, src); err != nil {
				return nil, err
			}
			if err := rejectSymlinkEscape(workdir, dst); err != nil {
				return nil, err
			}
			if err := rejectNonRegularFile(src); err != nil {
				return nil, err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return nil, err
			}
			if err := os.Rename(src, dst); err != nil {
				return nil, err
			}
			moved = append(moved, map[string]any{"from": src, "to": dst})
		case V4AOpUpdate:
			target, err := resolvePathWithinWorkdir(workdir, op.Path)
			if err != nil {
				return nil, err
			}
			if err := rejectSymlinkEscape(workdir, target); err != nil {
				return nil, err
			}
			if err := rejectNonRegularFile(target); err != nil {
				return nil, err
			}
			bs, err := os.ReadFile(target)
			if err != nil {
				return nil, err
			}
			orig := string(bs)
			hasTrailingNewline := strings.HasSuffix(orig, "\n")
			lines := splitPreserveEmpty(orig)
			for _, h := range op.Hunks {
				updated, ok := applyV4AHunk(lines, h)
				if !ok {
					return nil, fmt.Errorf("failed to apply hunk to %s (context=%q)", op.Path, h.ContextHint)
				}
				lines = updated
			}
			out := strings.Join(lines, "\n")
			if hasTrailingNewline && !strings.HasSuffix(out, "\n") {
				out += "\n"
			}
			if err := os.WriteFile(target, []byte(out), 0o644); err != nil {
				return nil, err
			}
			changed = append(changed, target)
		default:
			return nil, fmt.Errorf("unsupported operation: %s", op.Op)
		}
	}

	return map[string]any{
		"changed": changed,
		"added":   added,
		"deleted": deleted,
		"moved":   moved,
	}, nil
}

func splitPreserveEmpty(s string) []string {
	// strings.Split preserves trailing empty when separator at end.
	// But if s is empty, Split returns [""], which is fine for patch operations.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

func applyV4AHunk(lines []string, h V4AHunk) ([]string, bool) {
	var oldSeq []string
	var newSeq []string
	for _, ln := range h.Lines {
		switch ln.Prefix {
		case ' ':
			oldSeq = append(oldSeq, ln.Content)
			newSeq = append(newSeq, ln.Content)
		case '-':
			oldSeq = append(oldSeq, ln.Content)
		case '+':
			newSeq = append(newSeq, ln.Content)
		default:
			oldSeq = append(oldSeq, ln.Content)
			newSeq = append(newSeq, ln.Content)
		}
	}
	if len(oldSeq) == 0 {
		return lines, true
	}

	start := indexOfSeq(lines, oldSeq)
	if start < 0 {
		// Try forgiving match by trimming trailing empty line in both.
		trimOld := trimTrailingEmpty(oldSeq)
		if len(trimOld) != len(oldSeq) {
			start = indexOfSeq(lines, trimOld)
			if start >= 0 {
				oldSeq = trimOld
			}
		}
	}
	if start < 0 {
		// Try anchored search around context_hint (if any).
		hint := strings.TrimSpace(h.ContextHint)
		if hint != "" {
			hintAt := indexOfSubstring(lines, hint)
			if hintAt >= 0 {
				windowStart := hintAt - 80
				if windowStart < 0 {
					windowStart = 0
				}
				windowEnd := hintAt + 80
				if windowEnd > len(lines) {
					windowEnd = len(lines)
				}
				if off := indexOfSeq(lines[windowStart:windowEnd], oldSeq); off >= 0 {
					start = windowStart + off
				}
			}
		}
	}
	if start < 0 {
		// Whitespace-normalized match (best-effort).
		if off := indexOfSeqNormalized(lines, oldSeq); off >= 0 {
			start = off
		}
	}
	if start < 0 {
		return nil, false
	}
	out := make([]string, 0, len(lines)-len(oldSeq)+len(newSeq))
	out = append(out, lines[:start]...)
	out = append(out, newSeq...)
	out = append(out, lines[start+len(oldSeq):]...)
	return out, true
}

func trimTrailingEmpty(seq []string) []string {
	out := append([]string{}, seq...)
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func indexOfSeq(hay []string, needle []string) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		ok := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func normalizeLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func indexOfSeqNormalized(hay []string, needle []string) int {
	if len(needle) == 0 {
		return 0
	}
	need := make([]string, len(needle))
	for i := range needle {
		need[i] = normalizeLine(needle[i])
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		ok := true
		for j := range needle {
			if normalizeLine(hay[i+j]) != need[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func indexOfSubstring(lines []string, sub string) int {
	sub = strings.ToLower(strings.TrimSpace(sub))
	if sub == "" {
		return -1
	}
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), sub) {
			return i
		}
	}
	return -1
}

func readFirstNLines(path string, n int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := make([]string, 0, n)
	for s.Scan() {
		out = append(out, s.Text())
		if len(out) >= n {
			break
		}
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return strings.Join(out, "\n"), nil
}
