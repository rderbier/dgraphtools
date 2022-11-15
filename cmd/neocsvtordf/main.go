package main

/*
Read CSV file and RDF template to produce Dgraph RDF file and Schema draft
Schema draft is just the list of prediates found in the template and a list of detected dgraph.type mapping with associated list of predicates.
You want to set the correct type and indexes on the prediates.

*/

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

type PredSchema struct {
	predicatesMap map[string]string
	types         map[string]map[string]bool
}
type PredicateConfig struct {
	Type   string
	Format []string
}
type Config struct {
	Version    string
	Predicates map[string]PredicateConfig
	EdgeNodes  map[string]string
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
	configFileName    string
	rdfTemplates      []string
	r                 *regexp.Regexp
	rdfRegexp         *regexp.Regexp
	opRegexp          *regexp.Regexp
	rdfMap            map[string]string
	tripleList        []string
	uidTypeMap        map[string]string
	config            Config
)

func init() {
	initFlags()
	if configFileName != "" {
		rawConfig, err := ioutil.ReadFile(configFileName)
		if err != nil {
			log.Fatal(err)
		}
		if !json.Valid(rawConfig) {
			log.Fatal(errors.New("config file is not a valid json"))
		}
		json.Unmarshal(rawConfig, &config)

	}
}
func initFlags() {

	flag.StringVar(&fileName, "f", "", "neo4j csv file to process")
	flag.StringVar(&outFileName, "o", "", "output rdf file. default to stdout")
	flag.StringVar(&configFileName, "c", "", "config file")
	flag.StringVar(&outSchemaFileName, "s", "", "output schema file. default to stdout")

	flag.Parse()
	if fileName == "" {
		flag.Usage()
	}

}

func main() {
	var outfile *os.File
	var outschema *os.File

	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	outfile = os.Stdout
	if outFileName != "" {
		var err error
		outfile, err = os.OpenFile(outFileName, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)

		}
		defer outfile.Close()
	}
	var p *PredSchema
	p = newPredSchema()
	if outSchemaFileName != "" {

		outschema, err = os.OpenFile(outSchemaFileName, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer outschema.Close()
	}

	// read csv values using csv.Reader
	processCsv(f, outfile, p)

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

func cvslineToTriples(line []string, headers []string, indexOfStart int, config Config, p *PredSchema) ([]string, error) {
	var output []string
	var out string
	id := line[0]
	isNode := true
	indexOfType := indexOfStart + 2
	if indexOfType < len(line) {
		isNode = (line[indexOfType] == "")
	}
	if isNode {
		// create RDFs for dgraph.typ and each off the nom empty attributes
		labels := strings.Split(line[1], ":")
		for _, l := range labels[1:] {
			if l != "" {
				out = fmt.Sprintf(`<_:k_%s> <dgraph.type> "%s" .`, id, l)
				output = append(output, out)
			}
		}
		for i := 2; i <= indexOfStart-1; i++ {
			predicate := headers[i]
			if line[i] != "" {
				if strings.HasPrefix(line[i], "[") {
					// we have an array
					list := strings.Split(line[i][1:len(line[i])-1], "\"")
					for _, v := range list {
						if v != "" && v != "," {
							out = fmt.Sprintf(`<_:k_%s> <%s> "%s" .`, id, predicate, v)
							output = append(output, out)
						}
					}

				} else {
					val := line[i]
					if _, ok := config.Predicates[predicate]; ok {
						predicateType := config.Predicates[predicate].Type
						switch {
						case predicateType == "datetime":
							for _, f := range config.Predicates[predicate].Format {
								t, err := time.Parse(f, val)
								if err == nil {
									val = t.Format("2006-01-02T15:04:05Z")
									break
								}

							}
							break

						}
					}

					out = fmt.Sprintf(`<_:k_%s> <%s> "%s" .`, id, predicate, val)
					output = append(output, out)
				}
			}
		}
	} else if len(line) > indexOfStart {
		facets := ""
		predicate := line[indexOfType]

		if _, ok := config.EdgeNodes[predicate]; ok {
			//this edge is converted to an entity
			// we use the edge name as predicate name on start node
			// we use the map as entity type
			// we use <edge name>_to as predicate to end node
			entity_id := fmt.Sprintf("<_:r_%s_%s_%s>", line[indexOfStart], line[indexOfStart+1], predicate)
			out = fmt.Sprintf(`%s <dgraph.type> "%s" .`, entity_id, config.EdgeNodes[predicate])
			output = append(output, out)
			out = fmt.Sprintf(`%s <%s_to> <_:k_%s> .`, entity_id, predicate, line[indexOfStart+1])
			output = append(output, out)
			out = fmt.Sprintf(`<_:k_%s> <%s> %s .`, line[indexOfStart], predicate, entity_id)
			output = append(output, out)

		} else {
			var facetList []string
			for i := indexOfType + 1; i < len(line); i++ {
				if line[i] != "" {
					facetList = append(facetList, fmt.Sprintf(`%s="%s"`, headers[i], line[i]))
				}
			}
			if len(facetList) > 0 {
				facets = "(" + strings.Join(facetList, ",") + ")"
			}
			out = fmt.Sprintf(`<_:k_%s> <%s> <_:k_%s> %s .`, line[indexOfStart], line[indexOfType], line[indexOfStart+1], facets)
			output = append(output, out)
		}

	}

	return output, nil
}

func headersToMaps(headers []string) map[string]int {
	hmap := make(map[string]int)
	for i, h := range headers {
		hmap[h] = i
	}
	return hmap
}
func processCsv(f *os.File, o *os.File, p *PredSchema) {
	csvReader := csv.NewReader(f)
	headers, err := csvReader.Read()
	writer := bufio.NewWriter(o)
	defer writer.Flush()
	if err != nil {
		log.Fatal(err)
	}
	hmap := headersToMaps(headers)
	indexOfStart := hmap["_start"]
	// save all predicates
	for _, attr := range headers[2 : indexOfStart-1] {
		p.predicatesMap[attr] = "default"
	}
	i := 0
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err == nil {
			i += 1
			out, err := cvslineToTriples(rec, headers, indexOfStart, config, p)
			if err != nil {
				log.Fatal(errors.New(fmt.Sprintf("%s at  line %d", err.Error(), i)))
			}
			for _, l := range out {
				_, err := writer.WriteString(l + "\n")
				if err != nil {
					log.Fatalf("Got error while writing to a file. Err: %s", err.Error())
				}
			}

		} else {
			log.Print(err)
		}

	}
}
