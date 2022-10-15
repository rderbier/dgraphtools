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
	fileName          string
	templateFileName  string
	outFileName       string
	outSchemaFileName string
	rdfTemplates      []string
	r                 *regexp.Regexp
	rdfRegexp         *regexp.Regexp
	opRegexp          *regexp.Regexp
	rdfMap            map[string]string
	tripleList        []string
	predicateSchema   map[string]string
)

func init() {
	// rdfMap stores all single (S-P) triples created
	rdfMap = make(map[string]string)
	// predicateSchema stores all predicates with infered shcema
	predicateSchema = make(map[string]string)
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
	flag.StringVar(&outFileName, "o", "", "output rdf file. default to stdout")
	flag.StringVar(&outSchemaFileName, "s", "", "output schema file. default to stdout")

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
func loadSchema(f *os.File) {
	predRegexp, _ := regexp.Compile("([^ ]*)")

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		opMatch := predRegexp.FindAllString(line, -1)
		if len(opMatch) > 2 {
			predicateSchema[opMatch[0][0:len(opMatch[0])-1]] = strings.Join(opMatch[1:len(opMatch)-1], " ")
		}
	}
}
func main() {
	var outfile *os.File
	var outschema *os.File
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
	if outSchemaFileName != "" {
		file, err := os.Open(outSchemaFileName)
		if err == nil {
			loadSchema(file)
			file.Close()
		}

		outschema, err = os.OpenFile(outSchemaFileName, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer outschema.Close()
	}

	// read csv values using csv.Reader
	processCsv(f)

	dumpRdf(tripleList, rdfMap, outfile, outschema)

}
func dumpRdf(tripleList []string, rdfMap map[string]string, f *os.File, schemafile *os.File) {
	for key, value := range rdfMap {
		line := fmt.Sprintf("%s %s .\n", key, value)
		if f != nil {
			f.WriteString(line)
		} else {
			fmt.Printf(line)
		}
	}
	if f == nil {
		f = os.Stdout
	} else {
		fmt.Printf("rdf exported to %s\n", f.Name())
	}
	for _, s := range tripleList {
		f.WriteString(s)
	}
	if schemafile == nil {
		schemafile = os.Stdout
	} else {
		fmt.Printf("schema exported to %s", schemafile.Name())
	}

	for key, value := range predicateSchema {
		line := fmt.Sprintf("%s: %s .\n", key, value)
		schemafile.WriteString(line)
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
func substituteColumnValues(line []string, tripleTemplate string, hmap map[string]int) (string, error) {
	for {
		// analyse the template line
		columnAndFunc := r.FindStringSubmatch(tripleTemplate)

		if len(columnAndFunc) == 0 {
			break
		}
		info := strings.Split(columnAndFunc[1], ",")
		column := info[0]
		match := r.FindStringIndex(tripleTemplate)
		col, exist := hmap[column]
		if !exist {
			err := errors.New(fmt.Sprintf("%s is referencing a non existing column %s", tripleTemplate, column))
			return "", err
		}
		val := line[col]
		if val == "" {
			fmt.Printf("RDF %s ignored because of empty value for column %s", tripleTemplate, column)
			break
		}
		// remove [] which could cause a nested substitution
		val = strings.Replace(val, "[", "", -1)
		val = strings.Replace(val, "]", "", -1)
		for _, f := range info[1:] {
			switch f {
			case "nospace":
				val = strings.ReplaceAll(val, " ", "_")
			case "toUpper":
				val = strings.ToUpper(val)
			case "toLower":
				val = strings.ToLower(val)
			default:
				err := errors.New(fmt.Sprintf("unsupported transformer %s", f))
				return "", err
			}

		}

		tripleTemplate = fmt.Sprintf("%s%s%s", tripleTemplate[:match[0]], val, tripleTemplate[match[1]:])
	}
	return tripleTemplate, nil
}
func cvslineToTriples(index int, line []string, templates []string, hmap map[string]int) ([]string, error) {
	var output []string
	var err error
	for _, triple := range templates {
		// replace all [] blocks one by one, loop until none found.
		triple, err = substituteColumnValues(line, triple, hmap)
		// end loop until none function found
		if err != nil {
			return nil, err
		}
		triple, err = substituteFunction(triple)
		if err == nil {
			output = append(output, triple)
		} else {
			return nil, err
		}
	}
	return output, nil
}
func rdfToMapAndPredicates(rdfs []string) error {

	for _, s := range rdfs {
		// extract s P O . from triple
		elt := rdfRegexp.FindStringSubmatch(s)
		if len(elt) >= 5 {
			pred := elt[2][1 : len(elt[2])-1]
			obj := elt[3]
			predtype := "default"
			if strings.HasPrefix(obj, "<") {
				predtype = "uid"
			} else if strings.HasPrefix(obj, "\"{") {
				predtype = "geo @index(geo)"
			}
			if elt[4] == "*" { // multiple predicate possible
				predtype = "[" + predtype + "]"
				tripleList = append(tripleList, fmt.Sprintf("%s %s %s .\n", elt[1], pred, obj))
			} else {
				rdfMap[elt[1]+" "+elt[2]] = elt[3]
			}
			if !strings.HasPrefix(pred, "dgraph") {
				if current, exist := predicateSchema[pred]; exist {
					if current != predtype {
						return errors.New(fmt.Sprintf("type mistmach on predicate %s : found %s and %s", pred, current, predtype))
					}
				}
				predicateSchema[pred] = predtype
			}

		} else {
			if !strings.HasPrefix(s, "#") {
				return errors.New("Invalid RDF generated " + s)
			}
		}

	}
	return nil

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
			triples, err := cvslineToTriples(i, rec, rdfTemplates, hmap)
			if err != nil {
				log.Fatal(errors.New(fmt.Sprintf("%s at  line %d", err.Error(), i)))
			}
			err = rdfToMapAndPredicates(triples)
			if err != nil {
				log.Fatal(err)
			}

		} else {
			log.Print(err)
		}

	}
}
