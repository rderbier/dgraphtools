build: all
all : csvtordf rdfdatetree neocsvtordf	
csvtordf:
	go build -o ./bin ./cmd/csvtordf 
rdfdatetree:
	go build -o ./bin ./cmd/rdfdatetree 
neocsvtordf:
	go build -o ./bin ./cmd/neocsvtordf 