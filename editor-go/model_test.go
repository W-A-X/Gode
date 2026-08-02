package editor

import "testing"

func TestTextModel_Basic(t *testing.T) {
	m := NewTextModel("line1\nline2\nline3")
	if m.LineCount() != 3 {
		t.Fatalf("LineCount = %d, want 3", m.LineCount())
	}
	if got := m.LineContent(1); got != "line1" {
		t.Errorf("LineContent(1) = %q, want %q", got, "line1")
	}
	if got := m.LineMaxColumn(1); got != 6 {
		t.Errorf("LineMaxColumn(1) = %d, want 6", got)
	}
}

func TestTextModel_CRLF(t *testing.T) {
	m := NewTextModel("a\r\nb\r\nc")
	if m.LineCount() != 3 {
		t.Fatalf("LineCount = %d, want 3", m.LineCount())
	}
	if m.EOL() != "\r\n" {
		t.Errorf("EOL = %q, want CRLF", m.EOL())
	}
	if got := m.LineContent(2); got != "b" {
		t.Errorf("LineContent(2) = %q, want %q", got, "b")
	}
}

func TestTextModel_Empty(t *testing.T) {
	m := NewTextModel("")
	if m.LineCount() != 1 {
		t.Fatalf("LineCount = %d, want 1", m.LineCount())
	}
	if got := m.LineContent(1); got != "" {
		t.Errorf("LineContent(1) = %q, want empty", got)
	}
	if got := m.LineMaxColumn(1); got != 1 {
		t.Errorf("LineMaxColumn(1) = %d, want 1", got)
	}
}

func TestTextModel_ValueInRange(t *testing.T) {
	m := NewTextModel("abcdef\nghijkl")
	got := m.ValueInRange(Range{
		Start: Position{Line: 1, Column: 3},
		End:   Position{Line: 2, Column: 4},
	})
	if want := "cdef\nghi"; got != want {
		t.Errorf("ValueInRange = %q, want %q", got, want)
	}
}

func TestTextModel_OffsetAt(t *testing.T) {
	m := NewTextModel("ab\ncd\n")
	cases := []struct {
		pos    Position
		offset int
	}{
		{Position{1, 1}, 0},
		{Position{1, 3}, 2},
		{Position{2, 1}, 3},
		{Position{2, 3}, 5},
		{Position{3, 1}, 6},
	}
	for _, c := range cases {
		if got := m.OffsetAt(c.pos); got != c.offset {
			t.Errorf("OffsetAt(%+v) = %d, want %d", c.pos, got, c.offset)
		}
	}
}

func TestTextModel_PositionAt(t *testing.T) {
	m := NewTextModel("ab\ncd\n")
	cases := []struct {
		offset int
		pos    Position
	}{
		{0, Position{1, 1}},
		{2, Position{1, 3}},
		{3, Position{2, 1}},
		{5, Position{2, 3}},
		{6, Position{3, 1}},
		{100, Position{3, 1}},
	}
	for _, c := range cases {
		if got := m.PositionAt(c.offset); got != c.pos {
			t.Errorf("PositionAt(%d) = %+v, want %+v", c.offset, got, c.pos)
		}
	}
}

func TestSelection_Sorted(t *testing.T) {
	s := Selection{
		Anchor: Position{Line: 3, Column: 5},
		Active: Position{Line: 1, Column: 2},
	}
	r := s.Sorted()
	if r.Start != (Position{Line: 1, Column: 2}) || r.End != (Position{Line: 3, Column: 5}) {
		t.Errorf("Sorted() = %+v, want start{1,2} end{3,5}", r)
	}
}
