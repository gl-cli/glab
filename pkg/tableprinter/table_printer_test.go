package tableprinter

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_ttyTablePrinter_truncate(t *testing.T) {
	buf := bytes.Buffer{}
	tp := NewTablePrinter()
	tp.SetTTYSeparator(" ")
	tp.SetTerminalWidth(5)
	tp.SetIsTTY(true)

	tp.AddCell("1")
	tp.AddCell("hello")
	tp.EndRow()
	tp.AddCell("2")
	tp.AddCell("world")
	tp.EndRow()

	buf.Write(tp.Bytes())

	expected := "1 h...\n2 w...\n"
	if buf.String() != expected {
		t.Errorf("expected: %q, got: %q", expected, buf.Bytes())
	}
}

func Test_nonTTYTablePrinter_truncate(t *testing.T) {
	buf := bytes.Buffer{}
	tp := NewTablePrinter()
	tp.SetTerminalWidth(5)
	tp.SetIsTTY(false)

	tp.AddCell("1")
	tp.AddCell("hello")
	tp.EndRow()
	tp.AddCell("2")
	tp.AddCell("world")
	tp.EndRow()

	buf.Write(tp.Bytes())

	expected := "1\thello\n2\tworld\n"
	if buf.String() != expected {
		t.Errorf("expected: %q, got: %q", expected, buf.Bytes())
	}
}

func TestTablePrinter(t *testing.T) {
	t.Run("nil pointer cell value", func(t *testing.T) {
		var createdAt *time.Time
		var status *string
		updatedAt := time.Now().UTC()

		tp := NewTablePrinter()
		tp.AddRow("id:", 1)
		tp.AddRow("Status:", status)
		tp.AddRow("Created:", createdAt)
		tp.AddRow("Updated:", updatedAt)

		expected := fmt.Sprintf("id:\t1\nStatus:\t<nil>\nCreated:\t<nil>\nUpdated:\t%v\n", updatedAt)
		require.Equal(t, expected, tp.String())
	})
}
