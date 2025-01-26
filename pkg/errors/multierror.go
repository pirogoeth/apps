package errors

import (
	"fmt"
	"strings"
)

var _ error = (*MultiError)(nil)

type MultiError struct {
	errs []error
}

func (m *MultiError) Add(err error) {
	m.errs = append(m.errs, err)
}

func (m *MultiError) AsError() error {
	if len(m.errs) > 0 {
		return m
	}

	return nil
}

func (m *MultiError) Error() string {
	builder := new(strings.Builder)
	fmt.Fprintf(builder, "*** %d errors have occurred:\n\n", len(m.errs))
	fmt.Fprintf(builder, "%s\n", strings.Repeat("-", 40))
	for idx, err := range m.errs {
		fmt.Fprintf(builder, "| Error #%d\n|\n", idx)
		for _, errLine := range strings.Split(err.Error(), "\n") {
			fmt.Fprintf(builder, "| %s\n", errLine)
			fmt.Fprintf(builder, "\\%s\n", strings.Repeat("-", 39))
		}
	}

	return builder.String()
}
