package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCvslineToTriples(t *testing.T) {

	assert := assert.New(t)
	var config Config
	line := []string{"0", ":Person", "Sweden", "", "99", "", "Emil", "", ""}
	headers := []string{"_id", "_labels", "from", "hobby", "kloutScore", "learn", "name", "pet", "title", "_start", "_end", "_type", "rating", "since"}
	var p *PredSchema
	p = newPredSchema()
	indexOfStart := initPredicatesMap(headers, p)

	triple, _ := cvslineToTriples(line, headers, indexOfStart, &config, p)
	assert.Equal(triple[0], "<_:k_0> <dgraph.type> \"Person\" .")
	assert.Equal(triple[1], "<_:k_0> <from> \"Sweden\" .")
	assert.Equal(triple[2], "<_:k_0> <kloutScore> \"99\" .")
	assert.Equal(triple[3], "<_:k_0> <name> \"Emil\" .")
	line = []string{"5", ":Person", "Sweden", "[\"a\",\"b\",\"c\"]", "123", "", "tester", "", ""}
	triple, _ = cvslineToTriples(line, headers, indexOfStart, &config, p)
	assert.Equal(triple[2], "<_:k_5> <hobby> \"a\" .")
	assert.Equal(triple[3], "<_:k_5> <hobby> \"b\" .")
	assert.Equal(triple[4], "<_:k_5> <hobby> \"c\" .")
	// test edges
	headers = []string{"_id", "_labels", "born", "name", "released", "tagline", "title", "_start", "_end", "_type", "rating", "roles", "summary"}
	p = newPredSchema()
	indexOfStart = initPredicatesMap(headers, p)
	line = []string{"", "", "", "", "", "", "", "5", "121", "DIRECTED", "", "", ""}
	triple, _ = cvslineToTriples(line, headers, indexOfStart, &config, p)
	assert.Equal(triple[0], "<_:k_5> <DIRECTED> <_:k_121>  .")
	// edge with array of property

	config.EdgeNodes = map[string]string{"ACTED_IN": "ACTED_IN"}

	line = []string{"", "", "", "", "", "", "", "71", "161", "ACTED_IN", "", "[\"Hero Boy\",\"Father\",\"Conductor\",\"Hobo\",\"Scrooge\",\"Santa Claus\"]", ""}
	triple, _ = cvslineToTriples(line, headers, indexOfStart, &config, p)

	assert.Equal(triple[0], "<_:r_71_161_ACTED_IN> <dgraph.type> \"ACTED_IN\" .")
	assert.Equal(triple[1], "<_:r_71_161_ACTED_IN> <ACTED_IN_to> <_:k_161> .")
	assert.Equal(triple[2], "<_:k_71> <ACTED_IN> <_:r_71_161_ACTED_IN> .")

	triple, _ = cvslineToTriples(line, headers, indexOfStart, &config, p)
	assert.Equal(triple[0], "<_:r_71_161_ACTED_IN> <dgraph.type> \"ACTED_IN\" .")
	headers = []string{"_id", "_labels", "actions", "create_time", "description", "id", "name", "_start", "_end", "_type", "registration_date"}
	p = newPredSchema()
	indexOfStart = initPredicatesMap(headers, p)
	var pc PredicateConfig
	pc.Type = "datetime"
	pc.Format = []string{"2006-01-02T15:04:05.999999999Z[UTC]", "2006-01-02T15:04Z"}
	config.Predicates = make(map[string]PredicateConfig)
	config.Predicates["create_time"] = pc
	//= map[string]PredicateConfig{"create_time": {"Type": "datetime", "Format": "[\"2006-01-02T15:04:05.999999999Z[UTC]\",\"2006-01-02T15:04Z\"]"}}
	line = []string{"5", ":Application:Config", "", "2022-09-22T09:56:54.212775768Z[UTC]", "Your friendly neightborhood AI", "Rewf1br4SNeocIJefmaH7g", "mother"}
	triple, _ = cvslineToTriples(line, headers, indexOfStart, &config, p)
	assert.Equal(triple[2], "<_:k_5> <create_time> \"2022-09-22T09:56:54Z\" .")

}
