idgen
=====

[Go](https://golang.org) library for application-side generation of IDs for new database
records. There are multiple implementations to choose from, according to the system's
requirements. The most relevant is Twitter's Snowflake algorithm, but some simple
alternatives are good for tests and migrations, for example.

Advantages (not provided by all implementations):

- No contention in the database (sequence generation).
- Merging datasets does not require rewriting IDs. 
- Uses less space than UUIDs.
- No need to wait for synchronization during ID generation.
- Reproducibility in tests.

Installation
------------

```sh
go get github.com/carloslenz/idgen
```

Usage
-----

```go
node := os.Getenv("SNOWFLAKE_NODE")
nodeID, err := strconv.Atoi(node)

// ...

snowflake := idgen.NewSnowflake(nodeID)

id, err := snowflake.NewIDs(1)

// ...
```

Check [idgen.go](https://github.com/carloslenz/idgen/blob/master/idgen.go) for more.

Limitations
-----------

No distributed negotiatiation of nodeMasks yet (for Snowflake), so they need to be
hard-coded.

License
-------

MIT