package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstituteFunction(t *testing.T) {
	assert := assert.New(t)
	s1, err := substituteFunction("some text =geoloc(39.28347,-78.469379) more text")
	expected := "some text \"{\\\"type\\\":\\\"Point\\\",\\\"coordinates\\\":[39.28347,-78.469379]}\"^^<geo:geojson> more text"
	assert.Equal(s1, expected)
	s2, _ := substituteFunction("some text =randomDate(2022-09-01,2022-09-01) more text")
	assert.Equal(s2, "some text 2022-09-01 more text")
	s1, err = substituteFunction("some text =dummy(39.28347,-78.469379) more text")
	assert.Equal(err.Error(), "unsupported function dummy")

}
func TestCvslineToTriples(t *testing.T) {

	assert := assert.New(t)
	line := []string{"Value one", "Value two", "Value [three]"}
	cols := map[string]int{
		"COL1":  0,
		"COL2":  1,
		"COL.3": 2,
	}
	templates := []string{"<_:test> <is> \"text with [COL1] and [COL2]\" ."}

	triple, _ := cvslineToTriples(0, line, templates, cols)
	assert.Equal(triple[0], "<_:test> <is> \"text with Value one and Value two\" .")
	templates = []string{"<_:test> <is> \"text with [COL1,toUpper] and [COL2,toLower]\" ."}
	triple, _ = cvslineToTriples(0, line, templates, cols)
	assert.Equal(triple[0], "<_:test> <is> \"text with VALUE ONE and value two\" .")
	templates = []string{"<_:test> <is> \"text with [COL1,dummyFunc]\" ."}
	_, err := cvslineToTriples(0, line, templates, cols)
	assert.Equal(err.Error(), "unsupported transformer dummyFunc")
	assert.Equal(triple[0], "<_:test> <is> \"text with VALUE ONE and value two\" .")
	templates = []string{"<_:test> <is> \"text with [COL.3]\" ."}
	triple, err = cvslineToTriples(0, line, templates, cols)
	assert.Equal(triple[0], "<_:test> <is> \"text with Value three\" .")

}
func TestParseSchema(t *testing.T) {
	schema := []string{"name: default .", "geoloc: geo @index(geo) .", "type School {", "  name", "}"}
	p := parseSchema(schema)
	assert.Equal(t, p.predicatesMap["name"], "default")
	assert.Equal(t, p.predicatesMap["geoloc"], "geo @index(geo)")
	assert.Equal(t, p.types["School"]["name"], true)

}
func TestRdfToMapAndPredicates(t *testing.T) {
	lines := []string{"<_:test> <link> <_:obj> .", "<_:test> <link> \"value\" ."}
	p := newPredSchema()
	var m *map[string]string
	var err error
	m, err = rdfToMapAndPredicates(lines, p)
	assert.Equal(t, err.Error(), "type mismatch on predicate link : found uid and default")
	p = newPredSchema()
	lines = []string{"<_:test> <dgraph.type> \"School\" .", "<_:test> <link> \"value\" ."}
	m, err = rdfToMapAndPredicates(lines, p)
	assert.Equal(t, p.types["School"]["link"], true)
	lines = []string{
		"<_:test> <pred> \"value with \"quoted\" part\" .",
		"<_:School> <geoloc> \"{\\\"type\\\":\\\"Point\\\",\\\"coordinates\\\":[-78.469379,39.28347]}\"^^<geo:geojson> .",
		"<_:School> <title> \"value with back\\slash\" .",
	}
	m, err = rdfToMapAndPredicates(lines, p)
	assert.Equal(t, (*m)["<_:test> <pred>"], "\"value with \\\"quoted\\\" part\"")
	assert.Equal(t, (*m)["<_:School> <geoloc>"], "\"{\\\"type\\\":\\\"Point\\\",\\\"coordinates\\\":[-78.469379,39.28347]}\"^^<geo:geojson>")
	assert.Equal(t, (*m)["<_:School> <title>"], "\"value with back\\\\slash\"")

	assert.Equal(t, p.types["School"]["link"], true)

}
