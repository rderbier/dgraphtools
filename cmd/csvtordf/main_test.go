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
	line := []string{"Value one", "Value two"}
	cols := map[string]int{
		"COL1": 0,
		"COL2": 1,
	}
	templates := []string{"<_:test> <is> \"text with [COL1] and [COL2]\" ."}

	tripple, _ := cvslineToTriples(0, line, templates, cols)
	assert.Equal(tripple[0], "<_:test> <is> \"text with Value one and Value two\" .")
	templates = []string{"<_:test> <is> \"text with [COL1,toUpper] and [COL2,toLower]\" ."}
	tripple, _ = cvslineToTriples(0, line, templates, cols)
	assert.Equal(tripple[0], "<_:test> <is> \"text with VALUE ONE and value two\" .")

}
