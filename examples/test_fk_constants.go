//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mysql-parser/ast"
)

func main() {
	fmt.Printf("ReferOptionNoOption: %v\n", ast.ReferOptionNoOption)
	fmt.Printf("ReferOptionNoAction: %v\n", ast.ReferOptionNoAction)
	fmt.Printf("ReferOptionRestrict: %v\n", ast.ReferOptionRestrict)
	fmt.Printf("ReferOptionCascade: %v\n", ast.ReferOptionCascade)
	fmt.Printf("ReferOptionSetNull: %v\n", ast.ReferOptionSetNull)
	fmt.Printf("ReferOptionSetDefault: %v\n", ast.ReferOptionSetDefault)
}
