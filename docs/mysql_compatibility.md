# MySQL Compatibility Reference

This document provides a comprehensive overview of MySQL syntax support in the Mist database engine. Features are categorized by implementation status with detailed notes on limitations and usage.

## Legend

- ✅ **Fully Supported** - Complete implementation with all major features
- ⚠️ **Partially Supported** - Basic implementation with some limitations
- ❌ **Not Supported** - Not implemented
- 📝 **Parsed Only** - Syntax is accepted but not enforced/executed

---

## Data Definition Language (DDL)

### Table Operations

| Statement | Status | Features | Limitations |
|-----------|--------|----------|-------------|
| `CREATE TABLE` | ✅ | Full support with constraints, foreign keys, auto-increment | No temporary tables, partitioning |
| `ALTER TABLE ADD COLUMN` | ✅ | Add columns with constraints and defaults | |
| `ALTER TABLE DROP COLUMN` | ✅ | Remove columns, auto-drop related indexes | |
| `ALTER TABLE MODIFY COLUMN` | ✅ | Change column type with data conversion | |
| `ALTER TABLE CHANGE COLUMN` | ✅ | Rename and modify columns | |
| `DROP TABLE` | ✅ | Full support with foreign key constraint checking | Supports IF EXISTS clause |
| `TRUNCATE TABLE` | ✅ | Reset table data and auto-increment counter | Validates foreign key constraints |

### Index Operations

| Statement | Status | Features | Limitations |
|-----------|--------|----------|-------------|
| `CREATE INDEX` | ✅ | Single-column hash indexes | No composite, B-tree, or full-text indexes |
| `DROP INDEX` | ✅ | Remove indexes | |
| `SHOW INDEX FROM table` | ✅ | Display table indexes | |

---

## Data Manipulation Language (DML)

### INSERT Operations

| Feature | Status | Notes |
|---------|--------|-------|
| `INSERT INTO ... VALUES` | ✅ | Single and multiple row inserts |
| `INSERT INTO ... (columns) VALUES` | ✅ | Column-specific inserts |
| Auto-increment handling | ✅ | Automatic ID generation |
| Default value processing | ✅ | Including `CURRENT_TIMESTAMP` |
| `INSERT ... SELECT` | ❌ | Not implemented |
| `INSERT ... ON DUPLICATE KEY UPDATE` | ❌ | Not implemented |

### SELECT Operations

| Feature | Status | Notes |
|---------|--------|-------|
| Basic SELECT | ✅ | Column selection, wildcards (*) |
| WHERE clause | ✅ | Comparison and logical operators |
| JOIN operations | ✅ | INNER, LEFT, RIGHT, CROSS joins |
| Comma-separated tables | ✅ | Implicit cross joins |
| Subqueries in FROM | ✅ | Virtual table creation |
| LIMIT clause | ✅ | `LIMIT count` and `LIMIT offset, count` |
| ORDER BY | ⚠️ | Basic implementation, may vary |
| GROUP BY | ✅ | Full support with aggregate functions |
| HAVING | ✅ | Complete support with aggregate conditions |
| UNION | ⚠️ | Planned - requires complex parser integration |
| Window functions | ❌ | Not implemented |
| Common Table Expressions (WITH) | ❌ | Not implemented |

### UPDATE Operations

| Feature | Status | Notes |
|---------|--------|-------|
| Basic UPDATE | ✅ | SET clause with WHERE conditions |
| Arithmetic expressions | ✅ | +, -, *, / operators |
| Column references in expressions | ✅ | `SET col = col * 1.1` |
| ON UPDATE triggers | ✅ | Automatic execution |
| Multi-table UPDATE | ❌ | Not implemented |

### DELETE Operations

| Feature | Status | Notes |
|---------|--------|-------|
| Basic DELETE | ✅ | WHERE clause support |
| Foreign key cascading | ✅ | CASCADE, SET NULL, SET DEFAULT |
| Multi-table DELETE | ❌ | Not implemented |

---

## Data Types

### Numeric Types

| Type | Status | Mapping | Notes |
|------|--------|---------|-------|
| `TINYINT` | ✅ | INT | All integer types map to INT |
| `SMALLINT` | ✅ | INT | |
| `MEDIUMINT` | ✅ | INT | |
| `INT` | ✅ | INT | |
| `BIGINT` | ✅ | INT | |
| `DECIMAL(p,s)` | ✅ | DECIMAL | With precision and scale |
| `NUMERIC` | ✅ | DECIMAL | Alias for DECIMAL |
| `FLOAT` | ✅ | FLOAT | |
| `DOUBLE` | ✅ | FLOAT | Maps to FLOAT |
| `BIT` | ✅ | BOOL | Maps to boolean |

### String Types

| Type | Status | Mapping | Notes |
|------|--------|---------|-------|
| `CHAR(n)` | ✅ | VARCHAR | Fixed-length treated as variable |
| `VARCHAR(n)` | ✅ | VARCHAR | With length validation |
| `BINARY` | ⚠️ | VARCHAR | No binary handling |
| `VARBINARY` | ⚠️ | VARCHAR | No binary handling |
| `TINYTEXT` | ✅ | TEXT | All text types map to TEXT |
| `TEXT` | ✅ | TEXT | |
| `MEDIUMTEXT` | ✅ | TEXT | |
| `LONGTEXT` | ✅ | TEXT | |
| `ENUM('val1','val2')` | ✅ | ENUM | With value validation |
| `SET('val1','val2')` | ❌ | Not supported | |

### Date and Time Types

| Type | Status | Mapping | Notes |
|------|--------|---------|-------|
| `DATE` | ✅ | DATE | String-based storage |
| `TIME` | ❌ | Not supported | |
| `DATETIME` | ✅ | TIMESTAMP | Maps to TIMESTAMP |
| `TIMESTAMP` | ✅ | TIMESTAMP | With CURRENT_TIMESTAMP support |
| `YEAR` | ❌ | Not supported | |

### Other Types

| Type | Status | Notes |
|------|--------|-------|
| `BOOLEAN` | ✅ | Native boolean support |
| `JSON` | ❌ | Not supported |
| `BLOB` types | ❌ | Use TEXT as alternative |
| Spatial types | ❌ | Not supported |

---

## Functions and Operators

### Aggregate Functions

| Function | Status | Notes |
|----------|--------|-------|
| `COUNT(*)` | ✅ | Count all rows |
| `COUNT(column)` | ✅ | Count non-null values |
| `COUNT(DISTINCT column)` | ✅ | Count unique values |
| `SUM(column)` | ✅ | Numeric columns only |
| `AVG(column)` | ✅ | Numeric columns only |
| `MIN(column)` | ✅ | Any comparable type |
| `MAX(column)` | ✅ | Any comparable type |

### Arithmetic Operators

| Operator | Status | Context |
|----------|--------|---------|
| `+` | ✅ | UPDATE expressions |
| `-` | ✅ | UPDATE expressions |
| `*` | ✅ | UPDATE expressions |
| `/` | ✅ | UPDATE expressions |

### Comparison Operators

| Operator | Status | Context |
|----------|--------|---------|
| `=` | ✅ | WHERE clauses, JOIN conditions |
| `!=` / `<>` | ✅ | WHERE clauses |
| `<` | ✅ | WHERE clauses |
| `<=` | ✅ | WHERE clauses |
| `>` | ✅ | WHERE clauses |
| `>=` | ✅ | WHERE clauses |

### Logical Operators

| Operator | Status | Context |
|----------|--------|---------|
| `AND` | ✅ | WHERE clauses |
| `OR` | ✅ | WHERE clauses |
| `NOT` | ⚠️ | Limited support |

### Date/Time Functions

| Function | Status | Notes |
|----------|--------|-------|
| `CURRENT_TIMESTAMP` | ✅ | In DEFAULT and ON UPDATE |
| `NOW()` | ❌ | Use CURRENT_TIMESTAMP |
| Date arithmetic | ❌ | Not implemented |

---

## Constraints and Keys

### Column Constraints

| Constraint | Status | Notes |
|------------|--------|-------|
| `PRIMARY KEY` | ✅ | Uniqueness enforced |
| `UNIQUE` | ✅ | Duplicate prevention |
| `NOT NULL` | ✅ | Null validation |
| `AUTO_INCREMENT` | ✅ | Automatic value generation |
| `DEFAULT value` | ✅ | Including functions |
| `ON UPDATE` | ✅ | Trigger execution |
| `CHECK` | ❌ | Not supported |

### Table Constraints

| Constraint | Status | Implementation | Notes |
|------------|--------|----------------|-------|
| `FOREIGN KEY` | ✅ | Full referential integrity | |
| `CASCADE` | ✅ | Delete/update cascading | |
| `SET NULL` | ✅ | Set referencing columns to NULL | |
| `SET DEFAULT` | ✅ | Set to default values | |
| `RESTRICT` | ✅ | Prevent deletion/updates | |
| `NO ACTION` | ✅ | Same as RESTRICT | |

---

## Transaction Support

| Feature | Status | Notes |
|---------|--------|-------|
| `START TRANSACTION` | ✅ | Begin transaction |
| `BEGIN` | ✅ | Alias for START TRANSACTION |
| `COMMIT` | ✅ | Commit changes |
| `ROLLBACK` | ✅ | Rollback changes |
| Nested transactions | ✅ | Full support with proper nesting |
| `SAVEPOINT name` | ✅ | Create savepoints |
| `ROLLBACK TO SAVEPOINT name` | ✅ | Partial rollback |
| `RELEASE SAVEPOINT name` | ✅ | Remove savepoint |
| Transaction isolation levels | ❌ | Not implemented |
| `LOCK TABLES` | ❌ | Not implemented |

---

## JOIN Operations

| Join Type | Status | Syntax | Notes |
|-----------|--------|--------|-------|
| INNER JOIN | ✅ | `table1 JOIN table2 ON condition` | |
| LEFT JOIN | ✅ | `table1 LEFT JOIN table2 ON condition` | |
| RIGHT JOIN | ✅ | `table1 RIGHT JOIN table2 ON condition` | |
| CROSS JOIN | ✅ | `table1 CROSS JOIN table2` | |
| Comma syntax | ✅ | `table1, table2 WHERE condition` | Implicit cross join |
| FULL OUTER JOIN | ❌ | Not supported | |
| NATURAL JOIN | ❌ | Not supported | |

---

## Administrative Commands

### SHOW Commands

| Command | Status | Notes |
|---------|--------|-------|
| `SHOW TABLES` | ✅ | List all tables |
| `SHOW INDEX FROM table` | ✅ | Display table indexes |
| `SHOW COLUMNS FROM table` | ❌ | Use `DESCRIBE table` alternative |
| `SHOW CREATE TABLE table` | ❌ | Not implemented |
| `SHOW DATABASES` | ❌ | Single database only |

### Other Commands

| Command | Status | Notes |
|---------|--------|-------|
| `DESCRIBE table` | ❌ | Not implemented |
| `EXPLAIN query` | ❌ | Not implemented |
| User management | ❌ | No authentication system |
| `GRANT`/`REVOKE` | ❌ | No permission system |

---

## Performance and Optimization

### Indexing

| Feature | Status | Notes |
|---------|--------|-------|
| Hash indexes | ✅ | Single-column, equality lookups |
| Index-optimized queries | ✅ | Automatic usage in WHERE clauses |
| Composite indexes | ❌ | Not supported |
| Covering indexes | ❌ | Not supported |
| Full-text indexes | ❌ | Not supported |

### Query Optimization

| Feature | Status | Notes |
|---------|--------|-------|
| Index usage in WHERE | ✅ | Automatic optimization |
| Join optimization | ⚠️ | Basic nested loop joins |
| Query caching | ❌ | Not implemented |
| Statistics | ❌ | Not collected |

---

## Limitations and Considerations

### General Limitations

1. **In-memory only** - No persistent storage
2. **Single database** - No multi-database support
3. **No user management** - No authentication or authorization
4. **Limited concurrency** - Thread-safe but no advanced locking
5. **Memory usage** - All data held in RAM

### MySQL Compatibility Notes

1. **Case sensitivity** - Table and column names are case-insensitive
2. **String comparisons** - Basic string comparison, no collation support
3. **Type conversion** - Automatic where possible
4. **Error handling** - MySQL-style error messages where applicable

### Best Practices for Mist

1. **Use indexes** for frequently queried columns
2. **Limit data size** due to memory constraints  
3. **Use transactions** for data consistency
4. **Consider foreign keys** for referential integrity
5. **Test compatibility** before migrating from MySQL

---

## Migration from MySQL

### Supported Migrations

- Basic table structures with standard data types
- Simple CRUD operations
- Basic relationships with foreign keys
- Simple reporting queries with aggregates

### Manual Adjustments Needed

- Remove unsupported data types (JSON, spatial, etc.)
- Simplify complex queries (remove UNION, window functions)
- Replace stored procedures with application logic
- Remove user management and permissions
- Add explicit transactions where needed

### Testing Recommendations

1. **Schema validation** - Ensure all tables create successfully
2. **Data type testing** - Verify data conversion works correctly
3. **Query testing** - Test all application queries
4. **Performance testing** - Verify acceptable performance with expected data volumes
5. **Transaction testing** - Ensure ACID properties work as expected

---

*This compatibility reference is based on Mist database engine version 1.0.0. Features and limitations may change in future versions.*