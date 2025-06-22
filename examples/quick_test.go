//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mist/mist"
)

func main() {
	engine := mist.NewSQLEngine()
	engine.ImportSQLFile("test_data_real/001_create_tables.sql")
	engine.ImportSQLFile("test_data_real/002_insert_sample_data.sql")

	result, _ := engine.Execute("SELECT status FROM invoices")
	fmt.Println("Invoice statuses:")
	mist.PrintResult(result)
}
