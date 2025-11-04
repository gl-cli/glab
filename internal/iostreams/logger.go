package iostreams

import "fmt"

// LogError prints output to StdErr
func (s *IOStreams) LogError(msg ...any) {
	fmt.Fprintln(s.StdErr, msg...)
}

// LogErrorf formats according to a format specifier and writes to StdErr
func (s *IOStreams) LogErrorf(format string, a ...any) {
	fmt.Fprintf(s.StdErr, format, a...)
}

// LogInfo is just like Log but prints output to StdOut
func (s *IOStreams) LogInfo(a ...any) {
	fmt.Fprintln(s.StdOut, a...)
}

// LogInfof formats according to a format specifier and writes to StdOut
func (s *IOStreams) LogInfof(format string, a ...any) {
	fmt.Fprintf(s.StdOut, format, a...)
}
