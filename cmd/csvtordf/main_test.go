package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSubstituteFunction(t *testing.T) {
	assert := assert.New(t)
	s1, err := substituteFunction("some text =geoloc(39.28347,-78.469379) more text")
	expected := "some text \"{\\\"type\\\":\\\"Point\\\",\\\"coordinates\\\":[39.28347,-78.469379]}\"^^<geo:geojson> more text"
	assert.Equal(s1, expected)
	s1, err = substituteFunction("some text =dummy(39.28347,-78.469379) more text")
	assert.Equal(err.Error(), "unsupported function dummy")
	s1, err = substituteFunction("some text =randomDate(2022-09-01,2022-09-01) more text")
	assert.Equal(s1, "some text 2022-09-01 more text")

}
