package main

import (
	"database/sql"
	"encoding/json"
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
	tableFields map[string][]TableField
}

type TableField struct {
	Name     string
	DataType string
	Nullable bool
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s := strings.Split(r.URL.Path, "/")
	fmt.Println(s)
	if len(s) < 1 || len(s) > 3 { //Вот такой запрос не пройдет GET /$table/$id/ - это ошибка?
		http.Error(w, "Bad request URL", http.StatusInternalServerError)
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
		fmt.Println("MethodPost")
	case http.MethodDelete:
		h.DeleteRow(w, r)
	default:
		http.Error(w, "unknown method", http.StatusInternalServerError)
	}

}

// Обработчик для просто Get / и GET /$table?limit=5&offset=7
func (h *Handler) GetTable(w http.ResponseWriter, r *http.Request) {
	s := strings.Split(r.URL.Path, "/")[1:]

	if s[0] == "" { // GET /
		response := make(map[string]interface{})
		tables := make(map[string]interface{})
		tables["tables"] = h.tableNames
		response["response"] = tables
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	base := path.Base(r.URL.Path)

	if !slices.Contains(h.tableNames, base) {
		msg := "unknown table"
		http.Error(w, msg, http.StatusNotFound)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offset = 0
	} else {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var tableData []map[string]interface{}
	tableData, err = h.GetTableData(base, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//fmt.Println(reflect.TypeOf(tableData[0]["name"]))

	if err = json.NewEncoder(w).Encode(tableData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

// Обработчик для GET /$table/$id
func (h *Handler) GetRow(w http.ResponseWriter, r *http.Request) {
	//l := len(r.URL.Path)
	pathString := strings.Split(r.URL.Path, "/")[1:]
	tableName, idString := pathString[0], pathString[1]
	if !slices.Contains(h.tableNames, tableName) {
		http.Error(w, "unknown table name", http.StatusInternalServerError)
		return
	}
	//todo Проверка id что он точно норм
	var id int
	var err error
	if id, err = strconv.Atoi(idString); err != nil {
		http.Error(w, "id must be integer", http.StatusInternalServerError)
		return
	}
	rowData, err := h.GetRowData(tableName, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(rowData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println(rowData)

}

func (h *Handler) PutRow(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//todo тут ниче пока не сделано
	fmt.Println(string(body))

}

func (h *Handler) DeleteRow(w http.ResponseWriter, r *http.Request) {
	pathString := strings.Split(r.URL.Path, "/")[1:]
	tableName, idString := pathString[0], pathString[1]
	if !slices.Contains(h.tableNames, tableName) {
		http.Error(w, "has no such table", http.StatusInternalServerError)
		return
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.DeleteRowData(tableName, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
