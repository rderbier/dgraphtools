# dgraphtools
Some tools in Go for graph database Dgraph

## csvtordf - Convert CSV to RDF file
```
go env -w GO111MODULE=auto
make
./bin/csvtordf -h
```
Read a CSV file with headers line by line and use a template file to generate text for each line of the CSV.

In template file, [column name] are substituted by the corresponding value of the CSV. It can be used with any text. For Dgraph, we are using a template file containing RDF statements.



### data substitution
> [column name]
```
<_:State> <name> "[School.State]" .

```

### post processing
> [column name,processing function]
```
<_:City_[School.City,nospace]> <dgraph.type> "City" .
```
Available processing :
- nospace : replace spaces by _
- toUpper
- toLower 


### functions
You can use functions to generate data 
> =function(params)
Note that you can use
```
<_:School_[School.ID]> <geoloc> =geoloc([LAT],[LNG]) .
<_:Donation_[Donation.ID]> <day> "=randomDate(2020-01-01,2022-12-31)" .
```
Available functions :
- geoloc(lat,long) : generate a RDF value with geoloc json string
- randomDate(start,end) : generate a random date

### list
If a predicate can be repeated for the same entity id (list) then  mark it with a star * instead a dot at the end of the RDF template :

```
<_:Donor[Donor.ID]> <donor_donation> <_:Donation_[Donation.ID]> *
```

### best practices for Dgraph RDF
nospace is important for id.

