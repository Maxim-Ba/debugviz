package wire

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// FormatSummary returns a human-readable summary of planned injections.
func FormatSummary(result Result) string {
	parts := planSummary(result.Plan)
	if len(parts) == 0 {
		return result.Path
	}
	return fmt.Sprintf("%s (%s)", result.Path, strings.Join(parts, ", "))
}

// WriteDiff prints a unified diff for dry-run output.
func WriteDiff(w io.Writer, result Result) error {
	if !result.Changed {
		return nil
	}
	_, _ = fmt.Fprintf(w, "wire %s\n", FormatSummary(result))
	beforeLines := splitLines(string(result.Before))
	afterLines := splitLines(string(result.After))
	diff := unifiedDiff(result.Path, beforeLines, afterLines)
	if diff != "" {
		_, _ = fmt.Fprint(w, diff)
		if !strings.HasSuffix(diff, "\n") {
			_, _ = fmt.Fprintln(w)
		}
	}
	return nil
}

func splitLines(src string) []string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	if src == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(src, "\n"), "\n")
}

type diffOp struct {
	kind diffKind
	line string
}

type diffKind int

const (
	diffEqual diffKind = iota
	diffDelete
	diffInsert
)

func unifiedDiff(path string, before, after []string) string {
	ops := lineDiff(before, after)
	hunks := collectHunks(ops, 3)
	if len(hunks) == 0 {
		return ""
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "--- a/%s\n", path)
	fmt.Fprintf(&buf, "+++ b/%s\n", path)
	for _, hunk := range hunks {
		oldStart, oldCount, newStart, newCount := hunkHeader(ops, hunk.start, hunk.end)
		fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
		for _, op := range ops[hunk.start:hunk.end] {
			switch op.kind {
			case diffEqual:
				fmt.Fprintf(&buf, " %s\n", op.line)
			case diffDelete:
				fmt.Fprintf(&buf, "-%s\n", op.line)
			case diffInsert:
				fmt.Fprintf(&buf, "+%s\n", op.line)
			}
		}
	}
	return buf.String()
}

type hunkSpan struct {
	start int
	end   int
}

func collectHunks(ops []diffOp, context int) []hunkSpan {
	var hunks []hunkSpan
	i := 0
	for i < len(ops) {
		for i < len(ops) && ops[i].kind == diffEqual {
			i++
		}
		if i >= len(ops) {
			break
		}
		changeStart := i
		for i < len(ops) && ops[i].kind != diffEqual {
			i++
		}
		changeEnd := i

		start := changeStart - context
		if start < 0 {
			start = 0
		}
		end := changeEnd + context
		if end > len(ops) {
			end = len(ops)
		}
		hunks = append(hunks, hunkSpan{start: start, end: end})
	}
	return mergeHunks(hunks)
}

func mergeHunks(hunks []hunkSpan) []hunkSpan {
	if len(hunks) == 0 {
		return nil
	}
	out := []hunkSpan{hunks[0]}
	for _, h := range hunks[1:] {
		last := &out[len(out)-1]
		if h.start <= last.end {
			if h.end > last.end {
				last.end = h.end
			}
			continue
		}
		out = append(out, h)
	}
	return out
}

func hunkHeader(ops []diffOp, start, end int) (oldStart, oldCount, newStart, newCount int) {
	oldLine, newLine := 1, 1
	for idx := 0; idx < start; idx++ {
		switch ops[idx].kind {
		case diffDelete:
			oldLine++
		case diffInsert:
			newLine++
		case diffEqual:
			oldLine++
			newLine++
		}
	}
	oldStart, newStart = oldLine, newLine
	for idx := start; idx < end; idx++ {
		switch ops[idx].kind {
		case diffDelete:
			oldCount++
		case diffInsert:
			newCount++
		case diffEqual:
			oldCount++
			newCount++
		}
	}
	if oldCount == 0 {
		oldCount = 1
	}
	if newCount == 0 {
		newCount = 1
	}
	return oldStart, oldCount, newStart, newCount
}

func lineDiff(before, after []string) []diffOp {
	n, m := len(before), len(after)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if before[i] == after[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	ops := make([]diffOp, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		if before[i] == after[j] {
			ops = append(ops, diffOp{kind: diffEqual, line: before[i]})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffOp{kind: diffDelete, line: before[i]})
			i++
		} else {
			ops = append(ops, diffOp{kind: diffInsert, line: after[j]})
			j++
		}
	}
	for i < n {
		ops = append(ops, diffOp{kind: diffDelete, line: before[i]})
		i++
	}
	for j < m {
		ops = append(ops, diffOp{kind: diffInsert, line: after[j]})
		j++
	}
	return ops
}
