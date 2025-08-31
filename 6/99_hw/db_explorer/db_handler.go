package main

import (
	"database/sql"
	"fmt"
)

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

	var result []map[string]interface{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		dataPointers := make([]interface{}, len(columns))
		for i, rowField := range h.tableFields[tableName] {
			switch rowField.DataType {
			case "INT":
				if rowField.Nullable {
					d := sql.NullInt64{}
					dataPointers[i] = &d
				} else {
					var d int
					dataPointers[i] = &d
				}
			case "VARCHAR":
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

		fmt.Println(h.tableFields[tableName])

		err = rows.Scan(dataPointers...)
		if err != nil {
			return nil, err
		}
		rowData := make(map[string]interface{})
		for i := range dataPointers {
			rowData[columns[i]] = dataPointers[i]
		}

		result = append(result, rowData)
	}

	return result, nil

}

func (h *Handler) GetRowData(tableName string, id int) (map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tableName)
	tableFieldsInfo, ok := h.tableFields[tableName]
	if !ok {
		return nil, fmt.Errorf("has no table %s", tableName)
	}
	//todo разобраться с типами

	rowData := make([]interface{}, len(tableFieldsInfo)) //Где-то тут скорее всего нужно сделать нужные типы полей
	rowDataPtrs := make([]interface{}, len(tableFieldsInfo))
	for i := range len(tableFieldsInfo) {
		rowDataPtrs[i] = &rowData[i]
	}

	err := h.db.QueryRow(query, id).Scan(rowDataPtrs...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})

	for i, v := range rowData {
		result[tableFieldsInfo[i].Name] = v
	}

	return result, nil

}

func (h *Handler) DeleteRowData(tableName string, id int) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)
	result, err := h.db.Exec(query, id)
	if err != nil {
		return err
	}
	//todo тут тоже надо доделать почти все. но оно минимально робит, удаляет

	fmt.Println(result.RowsAffected())

	return nil
}
