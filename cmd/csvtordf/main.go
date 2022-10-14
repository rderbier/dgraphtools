package main

/*
Replicate dgraph live functionality

Read RDF file
send set command and keep uid of newly created entities in a map (blank node)
substitute blank node with knwon uid from the map
save uid map
accept uid map file
stop on error returned by dgraph client and display the last line number
accept line number to start : used to re-try at last line

*/

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	fileName         string
	templateFileName string
	outFileName      string
	rdfTemplates     []string
	r                *regexp.Regexp
	rdfRegexp        *regexp.Regexp
	opRegexp         *regexp.Regexp
	rdfMap           map[string]string
	predicates       []string
)

func init() {
	rdfMap = make(map[string]string)
	// create the regeexp used to extract substitution in the template
	// find square bracketed parts [...]
	r, _ = regexp.Compile("\\[([\\w .,|]+)\\]")
	// rdf just find the non-sapces groups
	rdfRegexp, _ = regexp.Compile("(<\\S+>)\\s+(<\\S+>)\\s+([\"<].*[\">])\\s+([.\\*])")
	opRegexp, _ = regexp.Compile("=(\\w+)\\(([^,)]+),?([^,)]+)?\\)")
}
func initFlags() {

	flag.StringVar(&fileName, "f", "", "csv file to process")
	flag.StringVar(&templateFileName, "t", "", "template file")
	flag.StringVar(&outFileName, "o", "", "output file. default to input filename with .rdf extension")

	flag.Parse()
	if (fileName == "") || (templateFileName == "") {
		flag.Usage()
		log.Fatal("input file and template missing.")
	}
	// read rdf template
	file, err := os.Open(templateFileName)
	if err != nil {
		fmt.Println(templateFileName)
		log.Fatal(err)

	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		rdfTemplates = append(rdfTemplates, scanner.Text())
	}
	if err != nil {
		log.Fatal(err)
	}

}

func loadUidsmap(filename string) {

}

func main() {
	var outfile *os.File
	initFlags()
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if outFileName != "" {
		var err error
		outfile, err = os.OpenFile(outFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)

		}
		defer outfile.Close()
	}

	// read csv values using csv.Reader
	processCsv(f)

	dumpRdf(predicates, rdfMap, outfile)

}
func dumpRdf(predicates []string, rdfMap map[string]string, f *os.File) {
	for key, value := range rdfMap {
		line := fmt.Sprintf("%s %s .\n", key, value)
		if f != nil {
			f.WriteString(line)
		} else {
			fmt.Printf(line)
		}
	}
	for _, s := range predicates {
		if f != nil {
			f.WriteString(s)
		} else {
			fmt.Printf(s)
		}
	}
	if f != nil {
		fmt.Printf("rdf exported to %s", f.Name())
	}

}
func substituteFunction(triple string) (string, error) {
	for {
		opMatch := opRegexp.FindStringSubmatch(triple)
		if len(opMatch) < 2 {
			break
		}
		match := opRegexp.FindStringIndex(triple)
		val := ""
		switch opMatch[1] {
		case "geoloc":
			val = fmt.Sprintf("\"{\\\"type\\\":\\\"Point\\\",\\\"coordinates\\\":[%s,%s]}\"^^<geo:geojson>", opMatch[2], opMatch[3])
		case "randomDate":
			format := "2006-01-02"
			t1, _ := time.Parse(format, opMatch[2])
			t2, _ := time.Parse(format, opMatch[3])

			days := int(math.Round(t2.Sub(t1).Hours() / 24))
			d := rand.Intn(days + 1)
			t1 = t1.AddDate(0, 0, d)
			val = t1.Format(format)
		default:
			return "", errors.New("unsupported function " + opMatch[1])
		}
		triple = fmt.Sprintf("%s%s%s", triple[:match[0]], val, triple[match[1]:])
	}
	return triple, nil
}
func cvslineToTriples(index int, line []string, templates []string, hmap map[string]int) ([]string, error) {
	var output []string
	var err error
	for _, triple := range templates {
		// replace all [] blocks one by one, loop until none found.
		for {
			// analyse the template line
			columnAndFunc := r.FindStringSubmatch(triple)

			if len(columnAndFunc) == 0 {
				break
			}
			info := strings.Split(columnAndFunc[1], ",")
			column := info[0]
			match := r.FindStringIndex(triple)

			val := line[hmap[column]]
			if val == "" {
				err = errors.New(fmt.Sprintf("Empty value line %d", index))
				break
			}
			for _, f := range info[1:] {
				switch f {
				case "nospace":
					val = strings.ReplaceAll(val, " ", "_")
				case "toUpper":
					val = strings.ToUpper(val)
				case "toLower":
					val = strings.ToLower(val)
				}

			}

			triple = fmt.Sprintf("%s%s%s", triple[:match[0]], val, triple[match[1]:])
		}
		// loop until none function found
		if err == nil {
			triple, err = substituteFunction(triple)
			if err == nil {
				output = append(output, triple)
			} else {
				break
			}
		} else {
			break
		}

	}
	return output, err
}
func rdfToMapAndPredicates(rdfs []string) {

	for _, s := range rdfs {
		// extract s P O . from triple
		elt := rdfRegexp.FindStringSubmatch(s)
		if len(elt) >= 5 {
			if elt[4] == "*" { // multiple predicate possible
				predicates = append(predicates, fmt.Sprintf("%s %s %s .\n", elt[1], elt[2], elt[3]))
			} else {
				rdfMap[elt[1]+" "+elt[2]] = elt[3]
			}

		} else {
			log.Fatal("Invalid RDF generated " + s)
		}

	}
	return

}
func getHeaderMap(csvReader *csv.Reader) map[string]int {

	headers, err := csvReader.Read()
	if err != nil {
		log.Fatal(err)
	}
	hmap := make(map[string]int)
	for i, h := range headers {
		hmap[h] = i
	}
	return hmap
}
func processCsv(f *os.File) {
	csvReader := csv.NewReader(f)
	hmap := getHeaderMap(csvReader)
	i := 0
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err == nil {
			i += 1
			triples, _ := cvslineToTriples(i, rec, rdfTemplates, hmap)
			rdfToMapAndPredicates(triples)

		} else {
			log.Print(err)
		}

	}
}
