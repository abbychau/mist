package mist

import (
	"fmt"
	"strings"
	"sync"
)

// IndexType represents different types of indexes
type IndexType int

const (
	HashIndex IndexType = iota
	CompositeIndex
	FullTextIndex
	// Future: BTreeIndex, etc.
)

func (it IndexType) String() string {
	switch it {
	case HashIndex:
		return "HASH"
	case CompositeIndex:
		return "COMPOSITE"
	case FullTextIndex:
		return "FULLTEXT"
	default:
		return "UNKNOWN"
	}
}

// Index represents a database index
type Index struct {
	Name        string
	TableName   string
	ColumnName  string   // For single-column indexes (backward compatibility)
	ColumnNames []string // For multi-column indexes (composite)
	Type        IndexType
	Data        map[interface{}][]int // value -> row indexes
	IsParsedOnly bool                 // True for indexes that are parsed but not functionally implemented
	mutex       sync.RWMutex
}

// NewIndex creates a new single-column index
func NewIndex(name, tableName, columnName string, indexType IndexType) *Index {
	isParsedOnly := indexType == CompositeIndex || indexType == FullTextIndex
	
	return &Index{
		Name:        name,
		TableName:   tableName,
		ColumnName:  columnName,
		ColumnNames: []string{columnName},
		Type:        indexType,
		Data:        make(map[interface{}][]int),
		IsParsedOnly: isParsedOnly,
	}
}

// NewCompositeIndex creates a new multi-column index
func NewCompositeIndex(name, tableName string, columnNames []string, indexType IndexType) *Index {
	isParsedOnly := indexType == CompositeIndex || indexType == FullTextIndex
	
	return &Index{
		Name:        name,
		TableName:   tableName,
		ColumnName:  strings.Join(columnNames, ","), // For backward compatibility
		ColumnNames: columnNames,
		Type:        indexType,
		Data:        make(map[interface{}][]int),
		IsParsedOnly: isParsedOnly,
	}
}

// AddEntry adds an entry to the index
func (idx *Index) AddEntry(value interface{}, rowIndex int) {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	// Skip operations for parsed-only indexes
	if idx.IsParsedOnly {
		return
	}

	// Normalize the value for consistent indexing
	normalizedValue := normalizeIndexValue(value)

	if _, exists := idx.Data[normalizedValue]; !exists {
		idx.Data[normalizedValue] = make([]int, 0)
	}
	idx.Data[normalizedValue] = append(idx.Data[normalizedValue], rowIndex)
}

// RemoveEntry removes an entry from the index
func (idx *Index) RemoveEntry(value interface{}, rowIndex int) {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	// Skip operations for parsed-only indexes
	if idx.IsParsedOnly {
		return
	}

	normalizedValue := normalizeIndexValue(value)

	if rowIndexes, exists := idx.Data[normalizedValue]; exists {
		// Remove the specific row index
		for i, ri := range rowIndexes {
			if ri == rowIndex {
				idx.Data[normalizedValue] = append(rowIndexes[:i], rowIndexes[i+1:]...)
				break
			}
		}

		// Remove the key if no more row indexes
		if len(idx.Data[normalizedValue]) == 0 {
			delete(idx.Data, normalizedValue)
		}
	}
}

// UpdateEntry updates an entry in the index (remove old, add new)
func (idx *Index) UpdateEntry(oldValue, newValue interface{}, rowIndex int) {
	idx.RemoveEntry(oldValue, rowIndex)
	idx.AddEntry(newValue, rowIndex)
}

// Lookup finds row indexes for a given value
func (idx *Index) Lookup(value interface{}) []int {
	idx.mutex.RLock()
	defer idx.mutex.RUnlock()

	// Parsed-only indexes don't provide actual lookup functionality
	if idx.IsParsedOnly {
		return nil
	}

	normalizedValue := normalizeIndexValue(value)

	if rowIndexes, exists := idx.Data[normalizedValue]; exists {
		// Return a copy to avoid race conditions
		result := make([]int, len(rowIndexes))
		copy(result, rowIndexes)
		return result
	}

	return nil
}

// RebuildIndex rebuilds the entire index from table data
func (idx *Index) RebuildIndex(table *Table) error {
	idx.mutex.Lock()
	defer idx.mutex.Unlock()

	// Clear existing data
	idx.Data = make(map[interface{}][]int)

	// Skip rebuild for parsed-only indexes
	if idx.IsParsedOnly {
		return nil
	}

	// For single-column indexes (backward compatibility)
	if len(idx.ColumnNames) == 1 {
		colIndex := table.GetColumnIndex(idx.ColumnNames[0])
		if colIndex == -1 {
			return fmt.Errorf("column %s does not exist in table %s", idx.ColumnNames[0], table.Name)
		}

		// Rebuild from all rows
		rows := table.GetRows()
		for i, row := range rows {
			value := row.Values[colIndex]
			normalizedValue := normalizeIndexValue(value)

			if _, exists := idx.Data[normalizedValue]; !exists {
				idx.Data[normalizedValue] = make([]int, 0)
			}
			idx.Data[normalizedValue] = append(idx.Data[normalizedValue], i)
		}
	}

	return nil
}

// normalizeIndexValue normalizes values for consistent indexing
func normalizeIndexValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// Convert all numeric types to float64 for consistent comparison
	switch v := value.(type) {
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case string:
		return strings.ToLower(v) // Case-insensitive string indexing
	case bool:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// IndexManager manages all indexes for the database
type IndexManager struct {
	indexes map[string]*Index // index name -> index
	mutex   sync.RWMutex
}

// NewIndexManager creates a new index manager
func NewIndexManager() *IndexManager {
	return &IndexManager{
		indexes: make(map[string]*Index),
	}
}

// CreateIndex creates a new single-column index
func (im *IndexManager) CreateIndex(name, tableName, columnName string, indexType IndexType, table *Table) error {
	return im.CreateCompositeIndex(name, tableName, []string{columnName}, indexType, table)
}

// CreateCompositeIndex creates a new multi-column index
func (im *IndexManager) CreateCompositeIndex(name, tableName string, columnNames []string, indexType IndexType, table *Table) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	// Check if index already exists
	if _, exists := im.indexes[strings.ToLower(name)]; exists {
		return fmt.Errorf("index %s already exists", name)
	}

	// Validate all columns exist
	for _, columnName := range columnNames {
		if table.GetColumnIndex(columnName) == -1 {
			return fmt.Errorf("column %s does not exist in table %s", columnName, tableName)
		}
	}

	// Create the index
	var index *Index
	if len(columnNames) == 1 {
		index = NewIndex(name, tableName, columnNames[0], indexType)
	} else {
		index = NewCompositeIndex(name, tableName, columnNames, indexType)
	}

	// Build the index from existing data (only for functional indexes)
	if err := index.RebuildIndex(table); err != nil {
		return fmt.Errorf("failed to build index: %v", err)
	}

	im.indexes[strings.ToLower(name)] = index
	return nil
}

// DropIndex removes an index
func (im *IndexManager) DropIndex(name string) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	if _, exists := im.indexes[strings.ToLower(name)]; !exists {
		return fmt.Errorf("index %s does not exist", name)
	}

	delete(im.indexes, strings.ToLower(name))
	return nil
}

// GetIndex retrieves an index by name
func (im *IndexManager) GetIndex(name string) (*Index, bool) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	index, exists := im.indexes[strings.ToLower(name)]
	return index, exists
}

// GetIndexesForTable returns all indexes for a specific table and column
func (im *IndexManager) GetIndexesForTable(tableName, columnName string) []*Index {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	var result []*Index
	for _, index := range im.indexes {
		if strings.EqualFold(index.TableName, tableName) &&
			(columnName == "" || strings.EqualFold(index.ColumnName, columnName)) {
			result = append(result, index)
		}
	}
	return result
}

// UpdateIndexes updates all relevant indexes when a row is modified
func (im *IndexManager) UpdateIndexes(tableName string, rowIndex int, oldRow, newRow *Row, table *Table) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	for _, index := range im.indexes {
		if strings.EqualFold(index.TableName, tableName) {
			// Update the index with the new row data
			// Find the column index in the table
			columnIndex := -1
			for i, col := range table.Columns {
				if strings.EqualFold(col.Name, index.ColumnName) {
					columnIndex = i
					break
				}
			}

			if columnIndex >= 0 {
				// Remove old value
				if oldRow != nil && columnIndex < len(oldRow.Values) {
					oldValue := oldRow.Values[columnIndex]
					if oldRowIndexes, exists := index.Data[oldValue]; exists {
						// Remove this row index from the old value
						for i, idx := range oldRowIndexes {
							if idx == rowIndex {
								index.Data[oldValue] = append(oldRowIndexes[:i], oldRowIndexes[i+1:]...)
								break
							}
						}
						// Remove the key if no more rows reference it
						if len(index.Data[oldValue]) == 0 {
							delete(index.Data, oldValue)
						}
					}
				}

				// Add new value
				if columnIndex < len(newRow.Values) {
					newValue := newRow.Values[columnIndex]
					if _, exists := index.Data[newValue]; !exists {
						index.Data[newValue] = []int{}
					}
					index.Data[newValue] = append(index.Data[newValue], rowIndex)
				}
			}
		}
	}
}

// AddRowToIndexes adds a new row to all relevant indexes
func (im *IndexManager) AddRowToIndexes(tableName string, rowIndex int, row Row, table *Table) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	for _, index := range im.indexes {
		if strings.EqualFold(index.TableName, tableName) {
			colIndex := table.GetColumnIndex(index.ColumnName)
			if colIndex != -1 && colIndex < len(row.Values) {
				index.AddEntry(row.Values[colIndex], rowIndex)
			}
		}
	}
}

// RemoveRowFromIndexes removes a row from all relevant indexes
func (im *IndexManager) RemoveRowFromIndexes(tableName string, rowIndex int, row Row, table *Table) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	for _, index := range im.indexes {
		if strings.EqualFold(index.TableName, tableName) {
			colIndex := table.GetColumnIndex(index.ColumnName)
			if colIndex != -1 && colIndex < len(row.Values) {
				index.RemoveEntry(row.Values[colIndex], rowIndex)
			}
		}
	}
}

// ListIndexes returns all index names
func (im *IndexManager) ListIndexes() []string {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	var names []string
	for name := range im.indexes {
		names = append(names, name)
	}
	return names
}

// ClearTableIndexes clears all index data for a table (used by TRUNCATE)
func (im *IndexManager) ClearTableIndexes(tableName string) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	for _, index := range im.indexes {
		if strings.EqualFold(index.TableName, tableName) {
			index.mutex.Lock()
			index.Data = make(map[interface{}][]int)
			index.mutex.Unlock()
		}
	}
}

// DropTableIndexes removes all indexes for a table (used by DROP TABLE)
func (im *IndexManager) DropTableIndexes(tableName string) {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	// Collect indexes to delete
	var indexesToDelete []string
	for name, index := range im.indexes {
		if strings.EqualFold(index.TableName, tableName) {
			indexesToDelete = append(indexesToDelete, name)
		}
	}

	// Delete the collected indexes
	for _, name := range indexesToDelete {
		delete(im.indexes, name)
	}
}
