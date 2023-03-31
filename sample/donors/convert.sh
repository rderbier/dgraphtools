# This is my first bash script
echo "Converting Donors file"
# clean file and copy schema - schema may be updated by the conversion
rm donors.rdf 
rm donors.schema
cp schemas/donors.schema .
# convert all csv in same rdf file
CSV_PATH="./csv/10schools"
../../bin/csvtordf -f=${CSV_PATH}/Schools.csv -t=./templates/SchoolTemplate.txt -o=donors.rdf -s=donors.schema
../../bin/csvtordf -f=${CSV_PATH}/Projects.csv -t=./templates/ProjectTemplate.txt -o=donors.rdf -s=donors.schema
../../bin/csvtordf -f=${CSV_PATH}/Donors.csv -t=./templates/DonorTemplate.txt -o=donors.rdf -s=donors.schema
../../bin/csvtordf -f=${CSV_PATH}/Donations.csv -t=./templates/DonationTemplate.txt -o=donors.rdf -s=donors.schema