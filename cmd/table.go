package main

import (
	"io"
	"text/tabwriter"
	"time"
)

const (
	tableMinimumWidth = 0
	tableTabWidth     = 0
	tablePadding      = 2
	tablePadCharacter = ' '
	tableFlags        = 0
)

const (
	// noTime is shown when there is no time.
	noTime = "-"
	// unsetTime is the nanosecond value meaning the time is unset.
	unsetTime = 0
	// secondsPartOfNanosecondTime is the seconds argument when passing only nanoseconds to time.Unix.
	secondsPartOfNanosecondTime = 0
)

func newTableWriter(output io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(output, tableMinimumWidth, tableTabWidth, tablePadding, tablePadCharacter, tableFlags)
}

func formatTime(nanoseconds int64) string {
	if nanoseconds == unsetTime {
		return noTime
	}
	return time.Unix(secondsPartOfNanosecondTime, nanoseconds).Format(time.RFC3339)
}
