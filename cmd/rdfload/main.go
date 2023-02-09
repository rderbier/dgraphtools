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
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"google.golang.org/grpc"
)

var argsWithoutProg []string
var file *os.File
var dg *dgo.Dgraph
var dgraphUrl = "https://blue-surf-630041.grpc.us-east-1.aws.cloud.dgraph.io:443"
var dgraphAPIkey = "Y2NkODc1MWEyNWYyYWFiNjc2ZmNmOGQwNzRmYzhkYjE="
var conn *grpc.ClientConn
var blockSize = 1
var uidsmap map[string]string
var uidsmapFileName = "uidsmap.txt"
var uidsmapFileNameIn = "uidsmap.txt"

func initDgraphClient(url string, key string) *dgo.Dgraph {
	var err error
	conn, err = dgo.DialCloud(dgraphUrl, dgraphAPIkey)
	if err != nil {
		log.Fatal("While trying to dial gRPC")
	}

	dg := dgo.NewDgraphClient(api.NewDgraphClient(conn))

	fmt.Println("connected!")
	return dg
}
func init() {

	//flag.StringVar(&fileName, "file", "", "rdf file path to import")
	argsWithoutProg = os.Args[1:]
	fileName := argsWithoutProg[0]
	// You can get individual args with normal indexing.
	if fileName == "" {
		fmt.Println("Missing parameter, provide file name!")
		os.Exit(1)
	}

	var err error
	file, err = os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}

	dg = initDgraphClient(dgraphUrl, dgraphAPIkey)
	uidsmap = make(map[string]string)
}
func sendRdf(rdf string, dg *dgo.Dgraph) (map[string]string, error) {
	if dg != nil {
		mu := &api.Mutation{
			CommitNow: true,
		}
		ctx := context.Background()
		mu.SetNquads = []byte(rdf)
		response, err := dg.NewTxn().Mutate(ctx, mu)
		if err != nil {
			return nil, err
		}
		// response.Uids is a map all created Uids for blank nodes
		// _:e1 will return an entry for "e1" giving the generated uid

		fmt.Println(response.String())
		return response.Uids, nil
	} else {
		return nil, nil
	}

}
func saveUidsmap(filename string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	defer f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Saving uids map to %s", filename)
	for key, value := range uidsmap {
		_, err := fmt.Fprintf(f, "%s %s\n", key, value)
		if err != nil {
			fmt.Println(err)
			return
		}

	}
}
func loadUidsmap(filename string) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		id := strings.Split(scanner.Text(), " ")
		uidsmap[id[0]] = id[1]
	}

}

func main() {
	defer file.Close()
	loadUidsmap(uidsmapFileNameIn)
	if conn != nil {
		defer conn.Close()
		ctx := context.Background()
		resp, err := dg.NewTxn().Query(ctx, `schema {}`)
		if err != nil {
			log.Fatal(err)
		}

		// resp.Json contains the schema query response.
		fmt.Println(string(resp.Json))
	}

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	var sb strings.Builder
	i := 0
	totalLines := 0

	for scanner.Scan() {
		i += 1
		line := scanner.Text()
		// extract id of blank node
		part1, predicateObject, _ := strings.Cut(line, ">")
		_, subject, _ := strings.Cut(part1, "<")
		_, id, isblank := strings.Cut(subject, "_:")

		if isblank {
			if uuid, ok := uidsmap[id]; ok {
				subject = uuid
			}
		} else {
			// we don't push custom id so let's make it a blank node if it's not known
			if uuid, ok := uidsmap[subject]; ok {
				subject = uuid
			} else {
				subject = "_:" + subject
			}
		}

		sb.WriteString(fmt.Sprintf("<%s> %s", subject, predicateObject))

		if i == blockSize {
			i = 0
			//fmt.Println(sb.String())
			uids, err := sendRdf(sb.String(), dg)
			if err != nil {
				fmt.Printf("Abnormal end at line %d\n", totalLines)
				log.Fatal(err)
			}
			for key, value := range uids {
				uidsmap[key] = value
			}
			fmt.Print(".")
			sb.Reset()
			totalLines += blockSize
		}
	}
	if i != 0 {
		//fmt.Println(sb.String())
		uids, err := sendRdf(sb.String(), dg)
		if err != nil {
			fmt.Printf("Abnormal end at line %d\n", totalLines)
			log.Fatal(err)
		}
		for key, value := range uids {
			uidsmap[key] = value
		}
		fmt.Println(".")
		totalLines += i
	}
	saveUidsmap(uidsmapFileName)
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}

	fmt.Printf("Processed %d lines and %d entities", totalLines, len(uidsmap))
}
