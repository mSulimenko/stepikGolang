package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
)

// Handler tableFields - мапа, в которой хранится название таблицы - слайс инфы про поля
type Handler struct {
	db          *sql.DB
	tableNames  []string
	tableFields map[string][]TableField //Мапа: Название таблицы - слайс (имя, тип данных, nullable)
	pkNames     map[string]string
}

type TableField struct {
	Name     string
	DataType string
	Nullable bool
}

type CR map[string]interface{}

var ErrRecordNotFound = errors.New("record not found")

type ResponseTables struct {
	Response map[string]interface{} `json:"response"`
}

type ResponseError struct {
	Error string `json:"error"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	s := strings.Split(r.URL.Path, "/")

	if len(s) < 1 || len(s) > 3 { //Вот такой запрос не пройдет GET /$table/$id/ - это ошибка?
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResponseError{
			Error: "Bad request URL",
		})
		return
	}
	s = s[1:]

	switch r.Method {
	case http.MethodGet:
		if len(s) == 1 {
			h.GetTable(w, r)
		} else {
			h.GetRow(w, r)
		}

	case http.MethodPut:
		h.PutRow(w, r)
	case http.MethodPost:
		h.PostRow(w, r)
	case http.MethodDelete:
		h.DeleteRow(w, r)
	default:
		http.Error(w, "unknown method", http.StatusInternalServerError)
	}

}

// Обработчик для просто Get / и GET /$table?limit=5&offset=7
func (h *Handler) GetTable(w http.ResponseWriter, r *http.Request) {
	s := strings.Split(r.URL.Path, "/")[1:]

	w.Header().Set("Content-Type", "application/json")

	if s[0] == "" { // GET /
		response := ResponseTables{
			Response: CR{
				"tables": h.tableNames,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ResponseError{
				Error: err.Error(),
			})
			return
		}
		return
	}

	base := path.Base(r.URL.Path)

	if !slices.Contains(h.tableNames, base) {
		w.WriteHeader(http.StatusNotFound)
		msg := "unknown table"
		json.NewEncoder(w).Encode(ResponseError{
			Error: msg,
		})
		return
	}

	var limit, offset int
	var err error

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limit = 5
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			limit = 5
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offset = 0
	} else {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			offset = 0
		}
	}

	var tableData []map[string]interface{}
	tableData, err = h.GetTableData(base, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{
			Error: err.Error(),
		})
		return
	}

	if err = json.NewEncoder(w).Encode(ResponseTables{Response: CR{"records": tableData}}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{
			Error: err.Error(),
		})
		return
	}

}

// Обработчик для GET /$table/$id
func (h *Handler) GetRow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pathString := strings.Split(r.URL.Path, "/")[1:]
	tableName, idString := pathString[0], pathString[1]
	if !slices.Contains(h.tableNames, tableName) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ResponseError{
			Error: "unknown table",
		})
		return
	}

	var id int
	var err error
	if id, err = strconv.Atoi(idString); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ResponseError{
			Error: err.Error(),
		})
		return
	}
	rowData, err := h.GetRowData(tableName, id)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(ResponseError{
			Error: err.Error(),
		})
		return
	}

	if err = json.NewEncoder(w).Encode(ResponseTables{Response: CR{"record": rowData}}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{
			Error: err.Error(),
		})
		return
	}

}

// PUT /$table
func (h *Handler) PutRow(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application-json")

	s := strings.Split(r.URL.Path, "/")[1:]
	if len(s) > 2 { //
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ResponseError{
			Error: "wrong url path",
		})
		return
	}
	tableName := s[0]
	if !slices.Contains(h.tableNames, tableName) {
		w.WriteHeader(http.StatusNotFound)
		msg := "unknown table"
		json.NewEncoder(w).Encode(ResponseError{
			Error: msg,
		})
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{
			Error: err.Error(),
		})
	}

	var data interface{}
	err = json.Unmarshal(body, &data)

	mapData := data.(map[string]interface{})
	fieldsQuery := ""
	dataQuery := ""
	var params []interface{}
	ctr := 0
	settedFields := make(map[string]bool)
	for _, v := range h.tableFields[tableName] {
		settedFields[v.Name] = false
	}

	for k, v := range mapData {
		if k == h.pkNames[tableName] {
			continue //Пропускаем установку id
		}
		isFieldEx := false
		for _, tField := range h.tableFields[tableName] { // Проверка шо есть такое поле
			if k == tField.Name {
				isFieldEx = true
				break
			}
		}
		if !isFieldEx {
			continue
		} // Для пропуска несуществующего поля(наверно, через slices.Contains было бы удобнее, но лень)

		if ctr != 0 {
			fieldsQuery += ", "
		}
		fieldsQuery += fmt.Sprintf("%v", k)
		settedFields[k] = true

		if ctr != 0 {
			dataQuery += ", "
		}
		params = append(params, v)
		dataQuery += "?"
		ctr++
	}

	for k, v := range settedFields {
		if v {
			continue
		}
		if ctr != 0 {
			fieldsQuery += ", "
		}
		fieldsQuery += fmt.Sprintf("%v", k)

		if ctr != 0 {
			dataQuery += ", "
		}
		dataQuery += "?"
		var tableField TableField
		for _, tf := range h.tableFields[tableName] {
			if tf.Name == k {
				tableField = tf
				break
			}
		}
		nilValue := h.getNilValue(tableField)
		params = append(params, nilValue)

		ctr++
	}

	query := fmt.Sprintf("INSERT INTO %v(%v) VALUES (%v)", tableName, fieldsQuery, dataQuery)
	result, err := h.db.Exec(query, params...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}

	err = json.NewEncoder(w).Encode(
		ResponseTables{Response: CR{h.pkNames[tableName]: id}},
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}
}

func (h *Handler) getNilValue(tField TableField) interface{} {

	switch tField.DataType {
	case "INT":
		if tField.Nullable {
			return sql.NullInt64{
				Int64: 0,
				Valid: false,
			}
		}
		return 0

	case "VARCHAR", "TEXT":
		if tField.Nullable {
			return sql.NullString{
				String: "",
				Valid:  false,
			}
		}
		return ""
	case "FLOAT":
		if tField.Nullable {
			return sql.NullFloat64{
				Float64: 0,
				Valid:   false,
			}
		}
		return 0
	}

	return nil
}

func (h *Handler) checkType(tField TableField, value interface{}) bool {
	var ok bool
	switch tField.DataType {
	case "INT":
		_, ok = value.(int)
		if tField.Nullable && value == nil {
			ok = true
		}

	case "VARCHAR", "TEXT":
		_, ok = value.(string)
		if tField.Nullable && value == nil {
			ok = true
		}
	case "FLOAT":
		_, ok = value.(float64)
		if tField.Nullable && value == nil {
			ok = true
		}
	}
	if ok {
		return true
	}
	return false

}

// POST /$table/$id - обновляет запись, данные приходят в теле запроса (POST-параметры)
func (h *Handler) PostRow(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application-json")
	s := strings.Split(r.URL.Path, "/")[1:]
	if len(s) != 2 {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: "incorrect url"})
		return
	}
	id, err := strconv.Atoi(s[1])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: "unvalid id"})
		return
	}
	tableName := s[0]
	_, err = h.GetRowData(tableName, id) //tableName, id	А НАДО ЛИ отдельный запрос к бд, ЕСЛИ ID УЖЕ ЕСТЬ??? 	//Тут был rowData
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	} // По идее нужно просто понять, есть ли такая запись, если что вернуть 404

	//Вчитать и преобразовать данные для изменения
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}
	var reqData interface{}
	err = json.Unmarshal(body, &reqData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}

	mapReqData := reqData.(map[string]interface{})

	query := fmt.Sprintf("UPDATE %v SET ", tableName)
	ctr := 0
	var params []interface{}

	for k, v := range mapReqData {
		if k == h.pkNames[tableName] {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponseError{
				Error: fmt.Sprintf("field %v have invalid type", h.pkNames[tableName])},
			)
			return
		} //Проверка шо не id
		for _, tField := range h.tableFields[tableName] {
			if tField.Name == k {
				if !h.checkType(tField, v) {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(ResponseError{
						Error: fmt.Sprintf("field %v have invalid type", k)})
					return
				}
			}
		} //Проверка шо нужный тип
		//Возможно нужна проверка, шо есть такое поле

		if ctr > 0 {
			query += ", "
		}
		query += fmt.Sprintf("%v = ?", k)
		params = append(params, v)
		ctr++
	}

	query += fmt.Sprintf(" WHERE %v = %v", h.pkNames[tableName], id)

	result, err := h.db.Exec(query, params...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}

	updated, err := result.RowsAffected()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}

	err = json.NewEncoder(w).Encode(ResponseTables{Response: CR{"updated": updated}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}

}

// DELETE /$table/$id - удаляет запись
func (h *Handler) DeleteRow(w http.ResponseWriter, r *http.Request) {
	pathString := strings.Split(r.URL.Path, "/")[1:]
	tableName, idString := pathString[0], pathString[1]
	if !slices.Contains(h.tableNames, tableName) {
		http.Error(w, "has no such table", http.StatusNotFound)
		return
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}

	deleted, err := h.DeleteRowData(tableName, id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}

	err = json.NewEncoder(w).Encode(ResponseTables{Response: CR{"deleted": deleted}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseError{Error: err.Error()})
		return
	}

}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	handler := &Handler{
		db: db,
	}

	err := handler.initTables()
	if err != nil {
		return nil, err
	}

	return handler, nil
}
