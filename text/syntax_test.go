package text

import (
	"strings"
	"testing"
)

func findSyntaxChunk(st StyledText, content string) (Chunk, bool) {
	for _, chunk := range st.Chunks {
		if strings.Contains(chunk.Text, content) {
			return chunk, true
		}
	}
	return Chunk{}, false
}

func TestHighlightCodeMergesPlainRunsLinearly(t *testing.T) {
	source := strings.Repeat("ordinary text ", 20_000)
	got := HighlightCode(source, "go", DefaultSyntaxStyle())
	if len(got.Chunks) != 1 || got.Chunks[0].Text != source {
		t.Fatalf("plain source not merged correctly: chunks=%d", len(got.Chunks))
	}
}

func TestHighlightSQLKeywordsAreCaseInsensitive(t *testing.T) {
	style := DefaultSyntaxStyle()
	got := HighlightCode("SELECT value FROM records", "sql", style)
	for _, word := range []string{"SELECT", "FROM"} {
		chunk, ok := findSyntaxChunk(got, word)
		if !ok || chunk.FG != style.Keyword.FG {
			t.Fatalf("%q not highlighted as an SQL keyword: %#v", word, chunk)
		}
	}
}

func TestHighlightCommentMarkersDependOnLanguage(t *testing.T) {
	style := DefaultSyntaxStyle()
	python := HighlightCode("x // 2", "python", style)
	chunk, ok := findSyntaxChunk(python, "//")
	if !ok || chunk.FG != style.Operator.FG {
		t.Fatalf("Python floor-division operator styled as comment: %#v", chunk)
	}

	sql := HighlightCode("SELECT 1 -- note", "sql", style)
	chunk, ok = findSyntaxChunk(sql, "-- note")
	if !ok || chunk.FG != style.Comment.FG {
		t.Fatalf("SQL line comment was not highlighted: %#v", chunk)
	}
	sql = HighlightCode("value // 2", "sql", style)
	chunk, ok = findSyntaxChunk(sql, "//")
	if !ok || chunk.FG != style.Operator.FG {
		t.Fatalf("SQL slash operator was styled as a comment: %#v", chunk)
	}
}
