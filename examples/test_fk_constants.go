//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/model"
)

func main() {
	fmt.Printf("ReferOptionNoOption: %v\n", model.ReferOptionNoOption)
	fmt.Printf("ReferOptionNoAction: %v\n", model.ReferOptionNoAction)
	fmt.Printf("ReferOptionRestrict: %v\n", model.ReferOptionRestrict)
	fmt.Printf("ReferOptionCascade: %v\n", model.ReferOptionCascade)
	fmt.Printf("ReferOptionSetNull: %v\n", model.ReferOptionSetNull)
	fmt.Printf("ReferOptionSetDefault: %v\n", model.ReferOptionSetDefault)
}
