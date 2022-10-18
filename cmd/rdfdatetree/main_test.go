package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDumpDateRDF(t *testing.T) {

	assert := assert.New(t)
	buf := new(bytes.Buffer)
	format := "2006-01-02"
	start, _ := time.Parse(format, "2022-09-01")
	end, _ := time.Parse(format, "2022-09-01")
	dumpDateRDF(buf, start, end)
	expected := `<_:tt2022-09-01> <yyyymmdd> "2022-09-01" .
<_:tt2022-09-01> <dgraph.type> "TimeTreeDay" .
<_:tt2022-09-01> <year> "2022" .
<_:tt2022-09-01> <month> "9" .
<_:tt2022-09-01> <dayOfMonth> "1" .
<_:tt2022-09-01> <dayOfWeek> "4" .
<_:tt2022-09-01> <week> "35" .
`
	assert.Equal(buf.String(), expected)

}
