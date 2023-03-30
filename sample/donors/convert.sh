# This is my first bash script
echo "Converting Donors file"
rm donors.rdf
../../bin/csvtordf -f=/Users/raph/Rwork/Schools-CA10.csv -t=./templates/SchoolTemplate.txt -o=donors.rdf -s=donors.schema
../../bin/csvtordf -f=/Users/raph/Rwork/Projects-CA10.csv -t=./templates/ProjectTemplate.txt -o=donors.rdf -s=donors.schema
../../bin/csvtordf -f=/Users/raph/Rwork/Donors-CA10.csv -t=./templates/DonorTemplate.txt -o=donors.rdf -s=donors.schema
../../bin/csvtordf -f=/Users/raph/Rwork/Donations-CA10.csv -t=./templates/DonationTemplate.txt -o=donors.rdf -s=donors.schema