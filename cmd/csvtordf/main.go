package main

/*
Read CSV file and RDF template to produce Dgraph RDF file and Schema draft
Schema draft is just the list of prediates found in the template and a list of detected dgraph.type mapping with associated list of predicates.
You want to set the correct type and indexes on the prediates.

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

type PredSchema struct {
	predicatesMap map[string]string
	types         map[string]map[string]bool
}

func newPredSchema() *PredSchema {
	p := PredSchema{}
	p.predicatesMap = make(map[string]string)
	p.types = make(map[string]map[string]bool)
	return &p
}

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
	uidTypeMap        map[string]string
)

func init() {
	// rdfMap stores all single (S-P) triples created
	rdfMap = make(map[string]string)
	uidTypeMap = make(map[string]string)
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
func loadSchema(f *os.File) *PredSchema {

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return parseSchema(lines)
}
func parseSchema(lines []string) *PredSchema {
	p := newPredSchema()
	var i int
	typename := ""
	for _, line := range lines {
		line := strings.Trim(line, " ")
		if !strings.HasPrefix(line, "#") {
			i = strings.Index(line, ":")
			if i > -1 {
				pred := line[0:i]

				p.predicatesMap[pred] = strings.Trim(strings.Trim(line[i+1:], "."), " ")
			} else {
				if strings.HasPrefix(line, "type ") {
					typename = strings.Split(line, " ")[1]
					p.types[typename] = make(map[string]bool)
				} else if strings.HasPrefix(line, "}") {
					typename = ""
				} else {
					if typename != "" {
						p.types[typename][line] = true
					}
				}

			}

		}
	}
	return p

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
	var p *PredSchema
	if outSchemaFileName != "" {
		file, err := os.Open(outSchemaFileName)
		if err == nil {
			p = loadSchema(file)
			file.Close()
		}

		outschema, err = os.OpenFile(outSchemaFileName, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer outschema.Close()
	} else {
		p = newPredSchema()
	}

	// read csv values using csv.Reader
	processCsv(f, p)

	dumpRdf(tripleList, rdfMap, outfile)
	dumpSchema(outschema, p)

}
func dumpRdf(tripleList []string, rdfMap map[string]string, f *os.File) {
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

}
func dumpSchema(schemafile *os.File, p *PredSchema) {
	if schemafile == nil {
		schemafile = os.Stdout
	} else {
		fmt.Printf("schema exported to %s", schemafile.Name())
	}

	for key, value := range p.predicatesMap {
		line := fmt.Sprintf("%s: %s .\n", key, value)
		schemafile.WriteString(line)
	}
	for dqltype, predmap := range p.types {
		line := fmt.Sprintf("type %s {\n", dqltype)
		schemafile.WriteString(line)
		for pred, _ := range predmap {
			schemafile.WriteString(fmt.Sprintf(" %s\n", pred))
		}
		schemafile.WriteString("}\n")
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
		if val == "" || val == "NULL" {
			fmt.Printf("RDF %s ignored because of empty or NULL value for column %s\n", tripleTemplate, column)
			tripleTemplate = ""
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
		triple = strings.Trim(triple, " ")
		// ignore empty lines and commented lines
		if triple != "" && !strings.HasPrefix(triple, "#") {
			triple, err = substituteColumnValues(line, triple, hmap)
			// end loop until none function found
			if err != nil {
				return nil, err
			}
			triple, err = substituteFunction(triple)
			if err == nil {
				if triple != "" {
					output = append(output, triple)
				}
			} else {
				return nil, err
			}
		}
	}
	return output, nil
}
func rdfToMapAndPredicates(rdfs []string, p *PredSchema) (*map[string]string, error) {

	for _, s := range rdfs {
		// extract s P O . from triple
		elt := rdfRegexp.FindStringSubmatch(s)
		if len(elt) >= 5 {
			subj := elt[1]
			pred := elt[2][1 : len(elt[2])-1]
			obj := elt[3]
			predtype := "default"
			if strings.HasPrefix(obj, "<") {
				predtype = "uid"
			} else if strings.HasPrefix(obj, "\"{") {
				predtype = "geo @index(geo)"
			} else if strings.HasPrefix(obj, "\"") {
				// need to print  \" instead of " and \\ instead of \
				obj = strings.Replace(obj[1:len(obj)-1], "\\", "\\\\", -1)
				obj = strings.Replace(obj, "\"", "\\\"", -1)
				obj = "\"" + obj + "\""
			}
			if elt[4] == "*" { // multiple predicate possible
				predtype = "[" + predtype + "]"
				tripleList = append(tripleList, fmt.Sprintf("%s <%s> %s .\n", elt[1], pred, obj))
			} else {
				rdfMap[subj+" "+elt[2]] = obj
			}
			if !strings.HasPrefix(pred, "dgraph") {
				//fmt.Println(`${subj ${pred} ${obj}`)
				if current, exist := p.predicatesMap[pred]; exist {
					if (current == "uid" || current == "[uid]") && !strings.HasPrefix(current, predtype) {
						return nil, errors.New(fmt.Sprintf("type mismatch on predicate %s : found %s and %s", pred, current, predtype))
					}
				} else {
					p.predicatesMap[pred] = predtype
				}
			}
			// save types
			if pred == "dgraph.type" {
				uidTypeMap[subj] = strings.Trim(obj, "\"")
			} else {
				if knowntype, ok := uidTypeMap[subj]; ok {
					if _, istype := p.types[knowntype]; !istype {
						p.types[knowntype] = make(map[string]bool)
					}
					p.types[knowntype][pred] = true
				}
			}

		} else {
			log.Println("Invalid RDF generated " + s)
		}

	}
	return &rdfMap, nil

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
func processCsv(f *os.File, p *PredSchema) {
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
			_, err = rdfToMapAndPredicates(triples, p)
			if err != nil {
				log.Fatal(err)
			}

		} else {
			log.Print(err)
		}

	}
}
