package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type DatabaseHandler struct {
	db *sql.DB
}

func NewDatabaseHandler(db *sql.DB) *DatabaseHandler {
	return &DatabaseHandler{db}
}

func (dh *DatabaseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := make([]string, 0)

	path = strings.Split(r.URL.Path, "/")[1:]

	//if len(path) < 0 {
	//	http.NotFound(w, r)
	//	return
	//}

	table := path[0]

	switch r.Method {
	case http.MethodGet:
		if path[0] == "" {
			log.Println("This is handleGetTables method!")
			dh.handleGetTables(w, r)
		} else if path[0] != "" && len(path) == 1 {
			log.Println("This is handleGetTableEntries method!")
			dh.handleGetTableEntries(w, r, table)
		} else if len(path) == 2 {
			id, err := strconv.Atoi(path[1])
			if err != nil {
				http.Error(w, "Invalid ID", http.StatusBadRequest)
				return
			}
			log.Println("This is handleGetTableEntry method!")
			dh.handleGetTableEntry(w, r, table, id)
		}
	case http.MethodPut:
		if len(path) == 1 {
			log.Println("This is handleCreateTableEntry method!")
			dh.handleCreateTableEntry(w, r, table)
		}
	case http.MethodPost:
		if len(path) == 2 {
			id, err := strconv.Atoi(path[1])
			log.Println("id:", id)
			if err != nil {
				http.Error(w, "Invalid ID", http.StatusBadRequest)
				return
			}
			log.Println("This is handleUpdateTableEntry method!")
			dh.handleUpdateTableEntry(w, r, table, id)
		}
	case http.MethodDelete:
		if len(path) == 2 {
			id, err := strconv.Atoi(path[1])
			if err != nil {
				http.Error(w, "Invalid ID", http.StatusBadRequest)
				return
			}
			log.Println("This is handleDeleteTableEntry method!")
			dh.handleDeleteTableEntry(w, r, table, id)
		}
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (dh *DatabaseHandler) handleGetTables(w http.ResponseWriter, r *http.Request) {
	rows, err := dh.db.Query("SHOW TABLES")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		err := rows.Scan(&table)
		if err != nil {
			log.Println(err)
			continue
		}
		tables = append(tables, table)
	}

	json.NewEncoder(w).Encode(tables)
}

func (dh *DatabaseHandler) handleGetTableEntries(w http.ResponseWriter, r *http.Request, table string) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		limit = 5
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	rows, err := dh.db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", table, limit, offset))
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result []map[string]interface{}
	columns, err := rows.Columns()
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		for i := range values {
			values[i] = new(interface{})
		}
		err = rows.Scan(values...)
		if err != nil {
			log.Println(err)
			continue
		}

		entry := make(map[string]interface{})
		for i, col := range columns {
			val := *(values[i].(*interface{}))

			// Обработка различных типов данных
			switch v := val.(type) {
			case []byte:
				entry[col] = string(v) // Преобразование []byte в строку
			case int64, float64:
				entry[col] = strconv.FormatFloat(v.(float64), 'f', -1, 64) // Преобразование числовых значений в строку
			default:
				entry[col] = val // Если тип неизвестен, просто добавляем в результат как есть
			}
		}
		result = append(result, entry)
	}

	json.NewEncoder(w).Encode(result)
}

func (dh *DatabaseHandler) handleGetTableEntry(w http.ResponseWriter, r *http.Request, table string, id int) {
	// Выполняем запрос и получаем результат
	log.Println("id:", id)
	rows, err := dh.db.Query(fmt.Sprintf("SELECT * FROM %s WHERE id=?", table), id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Проверяем, что есть хотя бы одна строка результата
	if !rows.Next() {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Получаем метаданные столбцов
	columns, err := rows.Columns()
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Получаем значения строки
	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(interface{})
	}
	err = rows.Scan(values...)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Формируем результат
	result := make(map[string]interface{})
	for i, col := range columns {
		val := *(values[i].(*interface{}))
		// Обработка различных типов данных
		switch v := val.(type) {
		case []byte:
			result[col] = string(v) // Преобразование []byte в строку
		case int64, float64:
			result[col] = strconv.FormatFloat(v.(float64), 'f', -1, 64) // Преобразование числовых значений в строку
		default:
			result[col] = val // Если тип неизвестен, просто добавляем в результат как есть
		}
	}

	// Отправляем результат
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		log.Println(err)
	}
}

func (dh *DatabaseHandler) handleCreateTableEntry(w http.ResponseWriter, r *http.Request, table string) {
	// Парсим данные из тела запроса
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Строим SQL-запрос для вставки данных в таблицу
	columns := make([]string, 0)
	placeholders := make([]string, 0)
	values := make([]interface{}, 0)

	for key, val := range r.Form {
		columns = append(columns, key)
		placeholders = append(placeholders, "?")
		values = append(values, val[0])
	}
	//log.Println("columns:", columns)
	//log.Println("placeholders:", placeholders)
	//log.Println("values:", values)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table, strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))
	//log.Println("query:", query)

	// Выполняем SQL-запрос
	_, err = dh.db.Exec(query, values...)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (dh *DatabaseHandler) handleUpdateTableEntry(w http.ResponseWriter, r *http.Request, table string, id int) {
	// Парсим данные из тела запроса
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Строим SQL-запрос для обновления данных в таблице
	sets := make([]string, 0)
	values := make([]interface{}, 0)
	for key, value := range r.Form {
		sets = append(sets, fmt.Sprintf("%s=?", key))
		values = append(values, value[0])
	}
	values = append(values, id)
	log.Println("sets:", sets)
	log.Println("values:", values)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id=?", table, strings.Join(sets, ", "))
	log.Println("query:", query)

	// Выполняем SQL-запрос
	_, err = dh.db.Exec(query, values...)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (dh *DatabaseHandler) handleDeleteTableEntry(w http.ResponseWriter, r *http.Request, table string, id int) {
	// Строим SQL-запрос для удаления записи из таблицы
	query := fmt.Sprintf("DELETE FROM %s WHERE id=?", table)

	// Выполняем SQL-запрос
	_, err := dh.db.Exec(query, id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	db, err := sql.Open("mysql",
		"user:mypassword@tcp(localhost:8765)/testdb?&charset=utf8&interpolateParams=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.Handle("/", NewDatabaseHandler(db))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
