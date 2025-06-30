//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mysql-parser"
	"github.com/abbychau/mysql-parser/ast"
	_ "github.com/abbychau/mysql-parser/parser_driver"
)

func main() {
	p := parser.New()
	sql := "CREATE TABLE test (id INT, parent_id INT, FOREIGN KEY (parent_id) REFERENCES parent(id) ON DELETE CASCADE ON UPDATE RESTRICT)"
	stmtNodes, _, err := p.ParseSQL(sql)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	stmt := stmtNodes[0].(*ast.CreateTableStmt)
	for _, constraint := range stmt.Constraints {
		if constraint.Tp == ast.ConstraintForeignKey {
			fmt.Printf("Foreign key constraint found\n")
			if constraint.Refer != nil {
				if constraint.Refer.OnDelete != nil {
					fmt.Printf("OnDelete: %v (type: %T)\n", constraint.Refer.OnDelete.ReferOpt, constraint.Refer.OnDelete.ReferOpt)
				}
				if constraint.Refer.OnUpdate != nil {
					fmt.Printf("OnUpdate: %v (type: %T)\n", constraint.Refer.OnUpdate.ReferOpt, constraint.Refer.OnUpdate.ReferOpt)
				}
			}
		}
	}
}
