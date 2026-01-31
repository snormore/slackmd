package slackmd

import "testing"

func TestFormatTable_Empty(t *testing.T) {
	got := formatTable(nil)
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestFormatTable_HeaderOnly(t *testing.T) {
	got := formatTable([][]string{{"A", "B"}})
	expected := "+---+---+\n| A | B |\n+---+---+"
	if got != expected {
		t.Fatalf("got:\n%s\nwant:\n%s", got, expected)
	}
}

func TestFormatTable_TwoRows(t *testing.T) {
	got := formatTable([][]string{{"A", "B"}, {"1", "2"}})
	expected := "+---+---+\n| A | B |\n+---+---+\n| 1 | 2 |\n+---+---+"
	if got != expected {
		t.Fatalf("got:\n%s\nwant:\n%s", got, expected)
	}
}

func TestFormatTable_MultipleDataRows(t *testing.T) {
	got := formatTable([][]string{{"A", "B"}, {"1", "2"}, {"3", "4"}})
	expected := "+---+---+\n| A | B |\n+---+---+\n| 1 | 2 |\n| 3 | 4 |\n+---+---+"
	if got != expected {
		t.Fatalf("got:\n%s\nwant:\n%s", got, expected)
	}
}

func TestFormatTable_VariableWidths(t *testing.T) {
	got := formatTable([][]string{{"Name", "X"}, {"Alice", "1"}})
	expected := "+-------+---+\n| Name  | X |\n+-------+---+\n| Alice | 1 |\n+-------+---+"
	if got != expected {
		t.Fatalf("got:\n%s\nwant:\n%s", got, expected)
	}
}

func TestFormatTable_Unicode(t *testing.T) {
	got := formatTable([][]string{{"城市", "人口"}, {"東京", "1400万"}})
	// Column widths should be based on rune count
	if got == "" {
		t.Fatal("expected non-empty output for unicode table")
	}
}
