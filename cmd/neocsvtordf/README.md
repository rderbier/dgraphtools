# neocsvtordf
This utility converts a Neo4J CSV export to a RDF file and a schema file that can be used with Dgraph import features (bulk load or dgraph live).

Neo4J CSV file contains nodes and edges information with following structure :
- first column contains the node id and is empty for edges
- second column contains the list of types for the nodes in the format :Type1:Type2
- following columns before the "_start" columns are nodes attributes.
- columns "_start", "_end" and "_type" are used for edges and give staring node id, ending node id and edge name.
- columns after "type" are edge attributes

The utility accepts a config file to configure the conversion using ``-c <filename>`` option.

Without config file, the utility :
- creates triples for the node types with dgraph.type predicate
- creates triples for each non empty property value using the property name as predicate name.
- creates multiple predicates if the value is an array
- creates a triple for each edges in the form of ``<start node> <edge type> <end node>``
- add facets for each  edge attribute

Using the config file, you can
- specify that an attribute is a datetime and the list of date formats used in the export.
For example if create_time is a node attribute and the export contains 2 formats we can specify :
```
{
    "version" : "1.0",
    "predicates" : {
        "create_time" : {
            "type":"datetime",
            "format":["2006-01-02T15:04:05.999999999Z[UTC]","2006-01-02T15:04Z"]
        }
    }
}
```
- specify that an edge must be translated into an entity instead of a predicate. This is usefull if the edge has many attributes or arrays and if you have complex queries on those edge attributes. In this case we transform 

    ``(start node)-[edge type]->(end node) ``
    
    into

    ``(start)-[edge type]->(relation node)-[<edge type>_to]->(end node)``

    **edge to node is not implemented yet**
