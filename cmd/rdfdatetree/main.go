package main

/*
Generate a list of Days RDF from start to end date
The schema is
yyyymmdd: datetime @index(year) .
ttyear: int @index(int) .
ttmonth: int @index(int) .
ttdayOfMonth: int @index(int) .
ttdayOfWeek: int @index(int) .
ttweek: int @index(int) .
type TimeTreeDay {
	yyyymmdd
	ttyear
	ttmonth
	ttdayOfMonth
	ttdayOfWeek
	ttweek
}


*/

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

var (
	outFileName string
	t1          time.Time
	t2          time.Time
)

func initFlags() {
	flag.StringVar(&outFileName, "o", "", "output rdf file. default to stdout")
	flag.Parse()
	tail := flag.Args()
	if len(tail) < 2 {
		log.Fatal("missing args")
	}
	format := "2006-01-02"
	t1, _ = time.Parse(format, tail[0])
	t2, _ = time.Parse(format, tail[1])

}
func dumpDateRDF(outfile io.Writer, start time.Time, end time.Time) {
	for {
		d := start.Format("2006-01-02")
		fmt.Fprintf(outfile, "<_:tt%s> <yyyymmdd> \"%s\" .\n", d, d)
		fmt.Fprintf(outfile, "<_:tt%s> <dgraph.type> \"TimeTreeDay\" .\n", d)
		fmt.Fprintf(outfile, "<_:tt%s> <year> \"%d\" .\n", d, start.Year())
		fmt.Fprintf(outfile, "<_:tt%s> <month> \"%d\" .\n", d, start.Month())
		fmt.Fprintf(outfile, "<_:tt%s> <dayOfMonth> \"%d\" .\n", d, start.Day())
		fmt.Fprintf(outfile, "<_:tt%s> <dayOfWeek> \"%d\" .\n", d, int(start.UTC().Weekday()))
		_, w := start.UTC().ISOWeek()
		fmt.Fprintf(outfile, "<_:%s> <week> \"%d\" .\n", d, w)

		start = start.AddDate(0, 0, 1)
		if start.After(end) {
			break
		}
	}
}
func main() {
	initFlags()
	var outfile *os.File
	outfile = os.Stdout

	if outFileName != "" {
		var err error
		outfile, err = os.OpenFile(outFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)

		}
		defer outfile.Close()
	}
	start := t1
	end := t2
	if !t2.After(t1) {
		start = t2
		end = t1

	}
	fmt.Printf("rdf exported to %s\n", outfile.Name())
	dumpDateRDF(outfile, start, end)

}
