package list

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/tableprinter"
)

// createTablePrinter creates a table printer for all given tokens with column headers and values aligned.
func createTablePrinter(tokens Tokens) *tableprinter.TablePrinter {
	table := tableprinter.NewTablePrinter()
	table.NonTTYSeparator = " "
	table.TTYSeparator = " "
	val := reflect.ValueOf(Token{})

	columnNames := make([]any, 0, val.Type().NumField())
	maxColumnWidths := make([]int, val.Type().NumField())

	for i := range val.Type().NumField() {
		field := val.Type().Field(i)
		maxColumnWidths[i] = len(field.Name) + 1
	}

	for _, token := range tokens {
		val := reflect.ValueOf(token)
		for i := range val.Type().NumField() {
			field := val.Field(i)

			length := len(field.Interface().(string))
			if length > maxColumnWidths[i] {
				maxColumnWidths[i] = length
			}
		}
	}

	for i := range val.Type().NumField() {
		field := val.Type().Field(i)
		columnNames = append(columnNames, fmt.Sprintf("%-*s", maxColumnWidths[i], toColumnName(field.Name)))
	}
	table.AddRow(columnNames...)

	for _, row := range tokens {
		val := reflect.ValueOf(row)
		values := make([]any, 0, val.Type().NumField())
		for i := range val.Type().NumField() {
			field := val.Field(i)
			values = append(values, fmt.Sprintf("%-*s", maxColumnWidths[i], field.Interface()))
		}
		table.AddRow(values...)
	}
	return table
}

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// change golang field name to a table column name
func toColumnName(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToUpper(snake)
}
