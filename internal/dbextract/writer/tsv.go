package writer

import "io"

// TSV uses CSV-style quoting with a tab delimiter to preserve embedded tabs
// and newlines without corrupting the column structure.
func NewTSV(out io.Writer) Writer { return newDelimited(out, '\t') }
