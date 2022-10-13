# dgraphtools
Some tools in Go for graph database Dgraph

## csvtordf - Convert CSV to RDF file
Read a CSV file with headers line by line and use a template file to generate text for each line of the CSV.

In template file, [column name] are substituted by the corresponding value of the CSV. 

```
go env -w GO111MODULE=auto
make
```

### data substitution
> [column name,processing function]
```
<_:State_[School.State,nospace]> <name> "[School.State]" .
<_:City_[School.City,nospace]> <dgraph.type> "City" .
```

### post processing
> [column name,processing function]

Available processing :
- nospace : replace spaces by _
- upper
- lower 
- anonymize
### functions
You can use functions to generate data 
> =function(params)
Note that you can use
```
<_:School_[School.ID]> <geoloc> =geoloc([LAT],[LNG]) .
```
Available functions :
- geoloc(lat,long) : generate a RDF value with geoloc json string
- randomDate(start,end) : generate a random date
