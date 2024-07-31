package dbconnector

// Credentials represents a user credentials
type Credentials struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

// Query represents a SQL query
type Query struct {
	SQL string `json:"sql"`
}

// QueryResult represents a sql query result:
//   - RowSet useful for queries that fetch data (usually 'select')
//   - AffectedRows is the number of rows affected by queries that write data (e.g. 'insert')
type QueryResult struct {
	RowSet       RowSet `json:"row_set"`
	AffectedRows int    `json:"affected_rows"`
	Error        error  `json:"error"`
}

// RowSet represents returned table data:
//   - Columns - column titles
//   - Values - list of rows, each as list of column values
type RowSet struct {
	Columns []string
	Rows    [][]any
}
