package mist

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// FunctionType represents different categories of built-in functions
type FunctionType int

const (
	FuncString FunctionType = iota
	FuncDateTime
	FuncMath
	FuncConditional
	FuncTypeConversion
)

// BuiltinFunction represents a built-in function implementation
type BuiltinFunction struct {
	Name     string
	Type     FunctionType
	MinArgs  int
	MaxArgs  int // -1 for unlimited
	Executor func(args []interface{}) (interface{}, error)
}

// Registry of all built-in functions
var builtinFunctions = map[string]*BuiltinFunction{
	// String Functions
	"CONCAT":    {Name: "CONCAT", Type: FuncString, MinArgs: 1, MaxArgs: -1, Executor: execConcat},
	"SUBSTRING": {Name: "SUBSTRING", Type: FuncString, MinArgs: 2, MaxArgs: 3, Executor: execSubstring},
	"LENGTH":    {Name: "LENGTH", Type: FuncString, MinArgs: 1, MaxArgs: 1, Executor: execLength},
	"UPPER":     {Name: "UPPER", Type: FuncString, MinArgs: 1, MaxArgs: 1, Executor: execUpper},
	"LOWER":     {Name: "LOWER", Type: FuncString, MinArgs: 1, MaxArgs: 1, Executor: execLower},
	"TRIM":      {Name: "TRIM", Type: FuncString, MinArgs: 1, MaxArgs: 1, Executor: execTrim},

	// Date/Time Functions
	"NOW":         {Name: "NOW", Type: FuncDateTime, MinArgs: 0, MaxArgs: 0, Executor: execNow},
	"CURDATE":     {Name: "CURDATE", Type: FuncDateTime, MinArgs: 0, MaxArgs: 0, Executor: execCurdate},
	"YEAR":        {Name: "YEAR", Type: FuncDateTime, MinArgs: 1, MaxArgs: 1, Executor: execYear},
	"MONTH":       {Name: "MONTH", Type: FuncDateTime, MinArgs: 1, MaxArgs: 1, Executor: execMonth},
	"DAY":         {Name: "DAY", Type: FuncDateTime, MinArgs: 1, MaxArgs: 1, Executor: execDay},
	"DATE_FORMAT": {Name: "DATE_FORMAT", Type: FuncDateTime, MinArgs: 2, MaxArgs: 2, Executor: execDateFormat},

	// Math Functions
	"ABS":     {Name: "ABS", Type: FuncMath, MinArgs: 1, MaxArgs: 1, Executor: execAbs},
	"ROUND":   {Name: "ROUND", Type: FuncMath, MinArgs: 1, MaxArgs: 2, Executor: execRound},
	"CEILING": {Name: "CEILING", Type: FuncMath, MinArgs: 1, MaxArgs: 1, Executor: execCeiling},
	"FLOOR":   {Name: "FLOOR", Type: FuncMath, MinArgs: 1, MaxArgs: 1, Executor: execFloor},
	"MOD":     {Name: "MOD", Type: FuncMath, MinArgs: 2, MaxArgs: 2, Executor: execMod},
	"POWER":   {Name: "POWER", Type: FuncMath, MinArgs: 2, MaxArgs: 2, Executor: execPower},

	// Conditional Functions
	"IF":        {Name: "IF", Type: FuncConditional, MinArgs: 3, MaxArgs: 3, Executor: execIf},
	"COALESCE":  {Name: "COALESCE", Type: FuncConditional, MinArgs: 1, MaxArgs: -1, Executor: execCoalesce},
	"IFNULL":    {Name: "IFNULL", Type: FuncConditional, MinArgs: 2, MaxArgs: 2, Executor: execIfnull},
	"NULLIF":    {Name: "NULLIF", Type: FuncConditional, MinArgs: 2, MaxArgs: 2, Executor: execNullif},

	// Type Conversion Functions
	"CAST":    {Name: "CAST", Type: FuncTypeConversion, MinArgs: 2, MaxArgs: 2, Executor: execCast},
	"CONVERT": {Name: "CONVERT", Type: FuncTypeConversion, MinArgs: 2, MaxArgs: 2, Executor: execConvert},
}

// GetBuiltinFunction returns a builtin function by name
func GetBuiltinFunction(name string) (*BuiltinFunction, bool) {
	fn, exists := builtinFunctions[strings.ToUpper(name)]
	return fn, exists
}

// ExecuteFunction executes a function call with the given arguments
func ExecuteFunction(funcName string, args []interface{}) (interface{}, error) {
	fn, exists := GetBuiltinFunction(funcName)
	if !exists {
		return nil, fmt.Errorf("unknown function: %s", funcName)
	}

	// Validate argument count
	if len(args) < fn.MinArgs {
		return nil, fmt.Errorf("function %s requires at least %d arguments, got %d", funcName, fn.MinArgs, len(args))
	}
	if fn.MaxArgs != -1 && len(args) > fn.MaxArgs {
		return nil, fmt.Errorf("function %s accepts at most %d arguments, got %d", funcName, fn.MaxArgs, len(args))
	}

	return fn.Executor(args)
}

// String Function Implementations

func execConcat(args []interface{}) (interface{}, error) {
	var result strings.Builder
	for _, arg := range args {
		if arg == nil {
			return nil, nil // MySQL behavior: CONCAT with NULL returns NULL
		}
		result.WriteString(fmt.Sprintf("%v", arg))
	}
	return result.String(), nil
}

func execSubstring(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}

	str := fmt.Sprintf("%v", args[0])
	start, err := toInt64(args[1])
	if err != nil {
		return nil, fmt.Errorf("SUBSTRING: invalid start position: %v", err)
	}

	// MySQL uses 1-based indexing
	startIndex := int(start - 1)
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex >= len(str) {
		return "", nil
	}

	if len(args) == 2 {
		// SUBSTRING(str, start) - return from start to end
		return str[startIndex:], nil
	}

	// SUBSTRING(str, start, length)
	length, err := toInt64(args[2])
	if err != nil {
		return nil, fmt.Errorf("SUBSTRING: invalid length: %v", err)
	}

	if length <= 0 {
		return "", nil
	}

	endIndex := startIndex + int(length)
	if endIndex > len(str) {
		endIndex = len(str)
	}

	return str[startIndex:endIndex], nil
}

func execLength(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}
	str := fmt.Sprintf("%v", args[0])
	return int64(len(str)), nil
}

func execUpper(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}
	str := fmt.Sprintf("%v", args[0])
	return strings.ToUpper(str), nil
}

func execLower(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}
	str := fmt.Sprintf("%v", args[0])
	return strings.ToLower(str), nil
}

func execTrim(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}
	str := fmt.Sprintf("%v", args[0])
	return strings.TrimSpace(str), nil
}

// Date/Time Function Implementations

func execNow(args []interface{}) (interface{}, error) {
	return time.Now().Format("2006-01-02 15:04:05"), nil
}

func execCurdate(args []interface{}) (interface{}, error) {
	return time.Now().Format("2006-01-02"), nil
}

func execYear(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}

	dateStr := fmt.Sprintf("%v", args[0])
	t, err := parseDateTime(dateStr)
	if err != nil {
		return nil, fmt.Errorf("YEAR: invalid date format: %v", err)
	}

	return int64(t.Year()), nil
}

func execMonth(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}

	dateStr := fmt.Sprintf("%v", args[0])
	t, err := parseDateTime(dateStr)
	if err != nil {
		return nil, fmt.Errorf("MONTH: invalid date format: %v", err)
	}

	return int64(t.Month()), nil
}

func execDay(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}

	dateStr := fmt.Sprintf("%v", args[0])
	t, err := parseDateTime(dateStr)
	if err != nil {
		return nil, fmt.Errorf("DAY: invalid date format: %v", err)
	}

	return int64(t.Day()), nil
}

func execDateFormat(args []interface{}) (interface{}, error) {
	if args[0] == nil || args[1] == nil {
		return nil, nil
	}

	dateStr := fmt.Sprintf("%v", args[0])
	formatStr := fmt.Sprintf("%v", args[1])

	t, err := parseDateTime(dateStr)
	if err != nil {
		return nil, fmt.Errorf("DATE_FORMAT: invalid date format: %v", err)
	}

	// Convert MySQL format specifiers to Go format
	goFormat := convertMySQLFormatToGo(formatStr)
	return t.Format(goFormat), nil
}

// Math Function Implementations

func execAbs(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}

	num, err := toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("ABS: invalid numeric value: %v", err)
	}

	return math.Abs(num), nil
}

func execRound(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}

	num, err := toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("ROUND: invalid numeric value: %v", err)
	}

	if len(args) == 1 {
		return math.Round(num), nil
	}

	// ROUND(num, decimals)
	decimals, err := toInt64(args[1])
	if err != nil {
		return nil, fmt.Errorf("ROUND: invalid decimal places: %v", err)
	}

	multiplier := math.Pow(10, float64(decimals))
	return math.Round(num*multiplier) / multiplier, nil
}

func execCeiling(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}

	num, err := toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("CEILING: invalid numeric value: %v", err)
	}

	return math.Ceil(num), nil
}

func execFloor(args []interface{}) (interface{}, error) {
	if args[0] == nil {
		return nil, nil
	}

	num, err := toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("FLOOR: invalid numeric value: %v", err)
	}

	return math.Floor(num), nil
}

func execMod(args []interface{}) (interface{}, error) {
	if args[0] == nil || args[1] == nil {
		return nil, nil
	}

	dividend, err := toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("MOD: invalid dividend: %v", err)
	}

	divisor, err := toFloat64(args[1])
	if err != nil {
		return nil, fmt.Errorf("MOD: invalid divisor: %v", err)
	}

	if divisor == 0 {
		return nil, nil // MySQL behavior: MOD by zero returns NULL
	}

	return math.Mod(dividend, divisor), nil
}

func execPower(args []interface{}) (interface{}, error) {
	if args[0] == nil || args[1] == nil {
		return nil, nil
	}

	base, err := toFloat64(args[0])
	if err != nil {
		return nil, fmt.Errorf("POWER: invalid base: %v", err)
	}

	exponent, err := toFloat64(args[1])
	if err != nil {
		return nil, fmt.Errorf("POWER: invalid exponent: %v", err)
	}

	return math.Pow(base, exponent), nil
}

// Conditional Function Implementations

func execIf(args []interface{}) (interface{}, error) {
	condition := args[0]
	trueValue := args[1]
	falseValue := args[2]

	if isTruthy(condition) {
		return trueValue, nil
	}
	return falseValue, nil
}

func execCoalesce(args []interface{}) (interface{}, error) {
	for _, arg := range args {
		if arg != nil {
			return arg, nil
		}
	}
	return nil, nil
}

func execIfnull(args []interface{}) (interface{}, error) {
	if args[0] != nil {
		return args[0], nil
	}
	return args[1], nil
}

func execNullif(args []interface{}) (interface{}, error) {
	if compareValues(args[0], args[1]) == 0 {
		return nil, nil
	}
	return args[0], nil
}

// Type Conversion Function Implementations

func execCast(args []interface{}) (interface{}, error) {
	value := args[0]
	targetType := fmt.Sprintf("%v", args[1])

	if value == nil {
		return nil, nil
	}

	switch strings.ToUpper(targetType) {
	case "CHAR", "VARCHAR", "TEXT":
		return fmt.Sprintf("%v", value), nil
	case "INT", "INTEGER", "BIGINT":
		return toInt64(value)
	case "DECIMAL", "FLOAT", "DOUBLE":
		return toFloat64(value)
	case "DATE":
		dateStr := fmt.Sprintf("%v", value)
		t, err := parseDateTime(dateStr)
		if err != nil {
			return nil, fmt.Errorf("CAST: cannot convert to DATE: %v", err)
		}
		return t.Format("2006-01-02"), nil
	case "DATETIME", "TIMESTAMP":
		dateStr := fmt.Sprintf("%v", value)
		t, err := parseDateTime(dateStr)
		if err != nil {
			return nil, fmt.Errorf("CAST: cannot convert to DATETIME: %v", err)
		}
		return t.Format("2006-01-02 15:04:05"), nil
	default:
		return nil, fmt.Errorf("CAST: unsupported target type: %s", targetType)
	}
}

func execConvert(args []interface{}) (interface{}, error) {
	// CONVERT is essentially the same as CAST in MySQL
	return execCast(args)
}

// Helper Functions

func toInt64(value interface{}) (int64, error) {
	if value == nil {
		return 0, fmt.Errorf("cannot convert NULL to integer")
	}

	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		str := fmt.Sprintf("%v", value)
		return strconv.ParseInt(str, 10, 64)
	}
}


func parseDateTime(dateStr string) (time.Time, error) {
	// Try common MySQL date/time formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date/time format: %s", dateStr)
}

func convertMySQLFormatToGo(mysqlFormat string) string {
	// Convert MySQL format specifiers to Go time format
	// This is a simplified implementation covering common cases
	replacements := map[string]string{
		"%Y": "2006", // 4-digit year
		"%y": "06",   // 2-digit year
		"%m": "01",   // Month (01-12)
		"%d": "02",   // Day (01-31)
		"%H": "15",   // Hour (00-23)
		"%i": "04",   // Minutes (00-59)
		"%s": "05",   // Seconds (00-59)
		"%M": "January", // Full month name
		"%b": "Jan",     // Abbreviated month name
		"%W": "Monday",  // Full weekday name
		"%a": "Mon",     // Abbreviated weekday name
	}

	result := mysqlFormat
	for mysql, goFmt := range replacements {
		result = strings.ReplaceAll(result, mysql, goFmt)
	}

	return result
}

// evaluateFunctionCall evaluates a function call expression
func evaluateFunctionCall(funcCall *ast.FuncCallExpr, table *Table, row Row) (interface{}, error) {
	funcName := funcCall.FnName.L

	// Evaluate arguments
	var args []interface{}
	for _, arg := range funcCall.Args {
		value, err := evaluateExpressionInRow(arg, table, row)
		if err != nil {
			return nil, fmt.Errorf("error evaluating function argument: %v", err)
		}
		args = append(args, value)
	}

	// Execute the function
	return ExecuteFunction(funcName, args)
}

// evaluateFunctionCallOnJoinResult evaluates a function call in JOIN context
func evaluateFunctionCallOnJoinResult(funcCall *ast.FuncCallExpr, joinResult *JoinResult, row []interface{}) (interface{}, error) {
	funcName := funcCall.FnName.L

	// Evaluate arguments
	var args []interface{}
	for _, arg := range funcCall.Args {
		value, err := evaluateExpressionOnJoinResult(arg, joinResult, row)
		if err != nil {
			return nil, fmt.Errorf("error evaluating function argument: %v", err)
		}
		args = append(args, value)
	}

	// Execute the function
	return ExecuteFunction(funcName, args)
}

// evaluateCaseExpression evaluates a CASE expression
func evaluateCaseExpression(caseExpr *ast.CaseExpr, table *Table, row Row) (interface{}, error) {
	// CASE expr WHEN value1 THEN result1 [WHEN value2 THEN result2 ...] [ELSE result] END
	var caseValue interface{}
	var err error
	
	if caseExpr.Value != nil {
		// Simple CASE: CASE expr WHEN value THEN result
		caseValue, err = evaluateExpressionInRow(caseExpr.Value, table, row)
		if err != nil {
			return nil, fmt.Errorf("error evaluating CASE value: %v", err)
		}
	}

	// Evaluate each WHEN clause
	for _, whenClause := range caseExpr.WhenClauses {
		var conditionMet bool

		if caseExpr.Value != nil {
			// Simple CASE: compare with case value
			whenValue, err := evaluateExpressionInRow(whenClause.Expr, table, row)
			if err != nil {
				return nil, fmt.Errorf("error evaluating WHEN expression: %v", err)
			}
			conditionMet = (compareValues(caseValue, whenValue) == 0)
		} else {
			// Searched CASE: evaluate condition as boolean
			conditionResult, err := evaluateWhereCondition(whenClause.Expr, table, row)
			if err != nil {
				return nil, fmt.Errorf("error evaluating WHEN condition: %v", err)
			}
			conditionMet = conditionResult
		}

		if conditionMet {
			// Return the THEN result
			return evaluateExpressionInRow(whenClause.Result, table, row)
		}
	}

	// If no WHEN clause matched, return ELSE result or NULL
	if caseExpr.ElseClause != nil {
		return evaluateExpressionInRow(caseExpr.ElseClause, table, row)
	}

	return nil, nil
}

// evaluateCaseExpressionOnJoinResult evaluates a CASE expression in JOIN context
func evaluateCaseExpressionOnJoinResult(caseExpr *ast.CaseExpr, joinResult *JoinResult, row []interface{}) (interface{}, error) {
	// CASE expr WHEN value1 THEN result1 [WHEN value2 THEN result2 ...] [ELSE result] END
	var caseValue interface{}
	var err error
	
	if caseExpr.Value != nil {
		// Simple CASE: CASE expr WHEN value THEN result
		caseValue, err = evaluateExpressionOnJoinResult(caseExpr.Value, joinResult, row)
		if err != nil {
			return nil, fmt.Errorf("error evaluating CASE value: %v", err)
		}
	}

	// Evaluate each WHEN clause
	for _, whenClause := range caseExpr.WhenClauses {
		var conditionMet bool

		if caseExpr.Value != nil {
			// Simple CASE: compare with case value
			whenValue, err := evaluateExpressionOnJoinResult(whenClause.Expr, joinResult, row)
			if err != nil {
				return nil, fmt.Errorf("error evaluating WHEN expression: %v", err)
			}
			conditionMet = (compareValues(caseValue, whenValue) == 0)
		} else {
			// Searched CASE: evaluate condition as boolean
			conditionResult, err := evaluateWhereConditionOnJoinResult(whenClause.Expr, joinResult, row)
			if err != nil {
				return nil, fmt.Errorf("error evaluating WHEN condition: %v", err)
			}
			conditionMet = conditionResult
		}

		if conditionMet {
			// Return the THEN result
			return evaluateExpressionOnJoinResult(whenClause.Result, joinResult, row)
		}
	}

	// If no WHEN clause matched, return ELSE result or NULL
	if caseExpr.ElseClause != nil {
		return evaluateExpressionOnJoinResult(caseExpr.ElseClause, joinResult, row)
	}

	return nil, nil
}