package main

import (
	"database/sql"
	"fmt"
)

// Для изначального присваиваиня типов для полей
func assignDataPointersForRows(tableFields []TableField, fieldsAmount int) []interface{} {
	dataPointers := make([]interface{}, fieldsAmount)
	for i, rowField := range tableFields { //Итерируемся по столбцам с доп. данными
		switch rowField.DataType { //По типу столбца
		case "INT":
			if rowField.Nullable {
				d := sql.NullInt64{}
				dataPointers[i] = &d
			} else {
				var d int
				dataPointers[i] = &d
			}
		case "VARCHAR", "TEXT":
			if rowField.Nullable {
				d := sql.NullString{}
				dataPointers[i] = &d
			} else {
				var d string
				dataPointers[i] = &d
			}
		case "FLOAT":
			if rowField.Nullable {
				d := sql.NullFloat64{}
				dataPointers[i] = &d
			} else {
				var d float64
				dataPointers[i] = &d
			}
		default:
		}
	}
	return dataPointers
}

// Для приведения уже отсканеных значений к нужному типу
func convertPointersForRow(dataPointers []interface{}, tableFields []TableField) (map[string]interface{}, error) {

	curRow := make(map[string]interface{})

	for i, pointerVal := range dataPointers {
		switch tableFields[i].DataType {
		case "INT":
			if tableFields[i].Nullable {
				v, ok := pointerVal.(*sql.NullInt64)
				if !ok {
					return nil, fmt.Errorf("cannot convert to NullInt64")
				}

				if v.Valid == true {
					curRow[tableFields[i].Name] = v.Int64
				} else {
					curRow[tableFields[i].Name] = nil
				}
			} else {
				v, ok := pointerVal.(*int)
				if !ok {
					return nil, fmt.Errorf("cannot convert to int")
				}
				curRow[tableFields[i].Name] = *v
			}

		case "VARCHAR", "TEXT":
			if tableFields[i].Nullable {
				v, ok := pointerVal.(*sql.NullString)
				if !ok {
					return nil, fmt.Errorf("cannot convert to NullString")
				}

				if v.Valid == true {
					curRow[tableFields[i].Name] = v.String
				} else {
					curRow[tableFields[i].Name] = nil
				}
			} else {
				v, ok := pointerVal.(*string)
				if !ok {
					return nil, fmt.Errorf("cannot convert to string")
				}
				curRow[tableFields[i].Name] = *v
			}
		case "FLOAT":
			if tableFields[i].Nullable {
				v, ok := pointerVal.(*sql.NullFloat64)
				if !ok {
					return nil, fmt.Errorf("cannot convert to NullInt64")
				}

				if v.Valid == true {
					curRow[tableFields[i].Name] = v.Float64
				} else {
					curRow[tableFields[i].Name] = nil
				}
			} else {
				v, ok := pointerVal.(*float64)
				if !ok {
					return nil, fmt.Errorf("cannot convert to float65")
				}
				curRow[tableFields[i].Name] = *v
			}
		default:
		}
	}
	return curRow, nil
}

func (h *Handler) initTables() error {
	rows, err := h.db.Query("SHOW TABLES")
	if err != nil {
		return err
	}
	defer rows.Close()

	tableNames := []string{}
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return err

		}
		tableNames = append(tableNames, name)
	}
	if rows.Err() != nil {
		return err
	}
	h.tableNames = tableNames

	err = h.initTableFields()
	if err != nil {
		return err
	}

	err = h.initPKNames()
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) initPKNames() error {
	pkNames := make(map[string]string)
	query := `
        SELECT COLUMN_NAME 
        FROM INFORMATION_SCHEMA.COLUMNS 
        WHERE TABLE_NAME = ? 
          AND TABLE_SCHEMA = DATABASE()
          AND COLUMN_KEY = 'PRI'
    `

	for _, tn := range h.tableNames {
		var pkName string

		err := h.db.QueryRow(query, tn).Scan(&pkName)
		if err != nil {
			return err
		}

		pkNames[tn] = pkName
	}
	h.pkNames = pkNames

	return nil

}

func (h *Handler) initTableFields() error {
	tableFields := make(map[string][]TableField)
	for _, tableName := range h.tableNames {
		query := fmt.Sprintf("SELECT * FROM %s LIMIT 0", tableName)
		rows, err := h.db.Query(query)
		if err != nil {
			return err
		}

		var columnTypes []*sql.ColumnType
		columnTypes, err = rows.ColumnTypes()
		if err != nil {
			return err
		}
		rows.Close()

		var columnFields []TableField

		for _, column := range columnTypes {
			nullable, ok := column.Nullable()
			if !ok {
				nullable = false
			}

			info := TableField{
				Name:     column.Name(),
				DataType: column.DatabaseTypeName(),
				Nullable: nullable,
			}
			columnFields = append(columnFields, info)
		}
		tableFields[tableName] = columnFields
	}
	h.tableFields = tableFields
	return nil
}

func (h *Handler) GetTableData(tableName string, limit, offset int) ([]map[string]interface{}, error) {

	query := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", tableName)
	rows, err := h.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableFields := h.tableFields[tableName] // Инфа по столбцам для сокращения записи
	dataPointers := assignDataPointersForRows(tableFields, len(h.tableFields[tableName]))

	var result []map[string]interface{}

	for rows.Next() {
		err = rows.Scan(dataPointers...)
		if err != nil {
			return nil, err
		}

		curRow, err := convertPointersForRow(dataPointers, tableFields)
		if err != nil {
			return nil, err
		}
		result = append(result, curRow)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *Handler) GetRowData(tableName string, id int) (map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE %v = ?", tableName, h.pkNames[tableName])
	tableFields, ok := h.tableFields[tableName]
	if !ok {
		return nil, fmt.Errorf("has no table %s", tableName)
	}

	dataPointers := assignDataPointersForRows(tableFields, len(h.tableFields[tableName]))

	err := h.db.QueryRow(query, id).Scan(dataPointers...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		} else {
			return nil, err
		}
	}
	curRow, err := convertPointersForRow(dataPointers, tableFields)
	if err != nil {
		return nil, err
	}

	return curRow, nil

}

func (h *Handler) DeleteRowData(tableName string, id int) (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)
	result, err := h.db.Exec(query, id)
	if err != nil {
		return 0, err
	}

	deleted, _ := result.RowsAffected()

	return deleted, nil
}
