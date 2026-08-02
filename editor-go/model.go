// Package editor implements the core rendering layer of a VS Code-style
// code editor in Go, rendered with the gogpu/ui toolkit.
//
// The architecture mirrors VS Code's editor split:
//
//	model     — the text buffer (ITextModel contract + in-memory TextModel)
//	viewmodel — pure layout/geometry: viewport, scrolling, coordinate conversion
//	view      — the gogpu/ui widget that draws the editor and handles input
//
// ITextModel is deliberately an interface so the renderer can be driven by any
// backend — a native Go buffer, or a bridge to the VS Code extension host /
// editor model over IPC. The renderer never depends on how the text is stored.
package editor

import "strings"

// Position is a location in the text model. Both Line and Column are 1-based,
// following the VS Code convention: column 1 is the position before the first
// character of a line, and LineMaxColumn(line) == len(line) + 1.
type Position struct {
	Line   int
	Column int
}

// IsValid reports whether the position is within the given model.
func (p Position) IsValid(m ITextModel) bool {
	return p.Line >= 1 && p.Line <= m.LineCount() && p.Column >= 1 && p.Column <= m.LineMaxColumn(p.Line)
}

// Range is a half-open interval [Start, End) of the text model.
type Range struct {
	Start Position
	End   Position
}

// Selection has an anchor (where the user started selecting) and an active
// end (the cursor). When anchor == active, the selection is a caret.
type Selection struct {
	Anchor Position
	Active Position
}

// Cursor returns the active end of the selection.
func (s Selection) Cursor() Position { return s.Active }

// IsEmpty reports whether the selection has zero width.
func (s Selection) IsEmpty() bool { return s.Anchor == s.Active }

// Sorted returns the range spanned by the selection, normalized so that
// Start precedes End.
func (s Selection) Sorted() Range {
	if s.Anchor.Line < s.Active.Line || (s.Anchor.Line == s.Active.Line && s.Anchor.Column <= s.Active.Column) {
		return Range{Start: s.Anchor, End: s.Active}
	}
	return Range{Start: s.Active, End: s.Anchor}
}

// ITextModel is the data contract between the renderer and any text backend.
//
// It mirrors the subset of VS Code's ITextModel that the view layer consumes.
// A remote implementation (e.g. bridging the VS Code extension host over IPC)
// only needs to satisfy these methods.
type ITextModel interface {
	// LineCount returns the number of lines in the model. Always >= 1.
	LineCount() int

	// LineContent returns the text of the 1-based line, without the EOL.
	LineContent(line int) string

	// LineMaxColumn returns the 1-based maximum column of the line, i.e.
	// the position just past the last character.
	LineMaxColumn(line int) int

	// ValueInRange returns the text covered by the half-open range.
	ValueInRange(r Range) string

	// OffsetAt converts a position to a 0-based character offset into
	// Value(), counting EOL characters.
	OffsetAt(pos Position) int

	// PositionAt converts a 0-based character offset back to a position.
	PositionAt(offset int) Position

	// EOL returns the line ending used by the model.
	EOL() string
}

// IEditableTextModel extends ITextModel with edit capability.
type IEditableTextModel interface {
	ITextModel
	// Edit replaces the half-open range with the given text. Text uses \n for
	// line separators regardless of the model's EOL setting.
	Edit(r Range, text string)
}

// TextModel is an in-memory IEditableTextModel backed by a slice of lines.
type TextModel struct {
	lines []string
	eol   string
}

// NewTextModel builds a TextModel from raw text. Lines may be separated by
// \n, \r\n or \r. The resulting model always has at least one line.
func NewTextModel(text string) *TextModel {
	eol := "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	} else if strings.Contains(text, "\r") && !strings.Contains(text, "\n") {
		eol = "\r"
	}

	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	var lines []string
	if normalized == "" {
		lines = []string{""}
	} else {
		lines = strings.Split(normalized, "\n")
	}

	return &TextModel{lines: lines, eol: eol}
}

// LineCount implements ITextModel.
func (m *TextModel) LineCount() int { return len(m.lines) }

// EOL implements ITextModel.
func (m *TextModel) EOL() string { return m.eol }

// LineContent implements ITextModel.
func (m *TextModel) LineContent(line int) string {
	if line < 1 || line > len(m.lines) {
		return ""
	}
	return m.lines[line-1]
}

// LineMaxColumn implements ITextModel.
func (m *TextModel) LineMaxColumn(line int) int {
	if line < 1 || line > len(m.lines) {
		return 1
	}
	return len([]rune(m.lines[line-1])) + 1
}

// ValueInRange implements ITextModel.
func (m *TextModel) ValueInRange(r Range) string {
	if r.Start.Line == r.End.Line {
		if r.Start.Line < 1 || r.Start.Line > len(m.lines) {
			return ""
		}
		line := []rune(m.lines[r.Start.Line-1])
		from := clamp(r.Start.Column-1, 0, len(line))
		to := clamp(r.End.Column-1, from, len(line))
		return string(line[from:to])
	}

	var b strings.Builder
	for line := r.Start.Line; line <= r.End.Line; line++ {
		if line < 1 || line > len(m.lines) {
			continue
		}
		content := []rune(m.lines[line-1])
		if line == r.Start.Line {
			from := clamp(r.Start.Column-1, 0, len(content))
			b.WriteString(string(content[from:]))
			b.WriteString(m.eol)
		} else if line == r.End.Line {
			to := clamp(r.End.Column-1, 0, len(content))
			b.WriteString(string(content[:to]))
		} else {
			b.WriteString(m.lines[line-1])
			b.WriteString(m.eol)
		}
	}
	return b.String()
}

// OffsetAt implements ITextModel. Offsets count Go runes, with each EOL
// counting as one offset unit (matching VS Code's UTF-16 model for BMP text).
func (m *TextModel) OffsetAt(pos Position) int {
	if pos.Line < 1 {
		pos.Line = 1
	}
	if pos.Line > len(m.lines) {
		pos.Line = len(m.lines)
	}

	offset := 0
	for line := 1; line < pos.Line; line++ {
		offset += len([]rune(m.lines[line-1])) + 1 // +1 for the EOL
	}

	content := []rune(m.lines[pos.Line-1])
	if pos.Column < 1 {
		return offset
	}
	if pos.Column > len(content)+1 {
		return offset + len(content)
	}
	return offset + pos.Column - 1
}

// PositionAt implements ITextModel.
func (m *TextModel) PositionAt(offset int) Position {
	if offset <= 0 {
		return Position{Line: 1, Column: 1}
	}

	remaining := offset
	for line := 1; line <= len(m.lines); line++ {
		contentLen := len([]rune(m.lines[line-1]))
		if remaining <= contentLen {
			return Position{Line: line, Column: remaining + 1}
		}
		remaining -= contentLen + 1 // content plus its EOL
		if remaining < 0 {
			// Landed exactly on the EOL: position is the start of the next line.
			return Position{Line: line + 1, Column: 1}
		}
	}

	last := len(m.lines)
	return Position{Line: last, Column: len([]rune(m.lines[last-1])) + 1}
}

// Edit implements IEditableTextModel.
func (m *TextModel) Edit(r Range, text string) {
	start := r.Start
	end := r.End
	// Clamp to valid positions.
	if start.Line < 1 {
		start.Line = 1
		start.Column = 1
	}
	if end.Line > m.LineCount() {
		end.Line = m.LineCount()
		end.Column = m.LineMaxColumn(end.Line)
	}

	startCol := clamp(start.Column-1, 0, len([]rune(m.lines[start.Line-1])))
	endCol := clamp(end.Column-1, 0, len([]rune(m.lines[end.Line-1])))

	// Split the replacement text into lines (protocol always uses \n).
	insertLines := strings.Split(text, "\n")

	// Build prefix from start line: chars before startCol.
	startRunes := []rune(m.lines[start.Line-1])
	prefix := string(startRunes[:startCol])

	// Build suffix from end line: chars from endCol onward.
	endRunes := []rune(m.lines[end.Line-1])
	suffix := string(endRunes[endCol:])

	// Replace the range [start.Line .. end.Line] with the new lines.
	newLines := make([]string, 0, len(m.lines)-end.Line+start.Line+len(insertLines))
	// Lines before start.
	newLines = append(newLines, m.lines[:start.Line-1]...)

	if len(insertLines) == 1 {
		// Single-line replacement: prefix + text + suffix on one line.
		newLines = append(newLines, prefix+insertLines[0]+suffix)
	} else {
		// First new line = prefix + first insert line.
		newLines = append(newLines, prefix+insertLines[0])
		// Middle new lines.
		newLines = append(newLines, insertLines[1:len(insertLines)-1]...)
		// Last new line = last insert line + suffix.
		newLines = append(newLines, insertLines[len(insertLines)-1]+suffix)
	}

	// Lines after end.
	newLines = append(newLines, m.lines[end.Line:]...)

	m.lines = newLines
	if len(m.lines) == 0 {
		m.lines = []string{""}
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
