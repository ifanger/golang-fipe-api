package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type TableReference struct {
	Code  int    `json:"Codigo"`
	Month string `json:"Mes"`
}

func main() {
	port := 8080

	fmt.Printf("Starting server at %d\n", port)

	db, err := sql.Open("sqlite3", "fipe.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	createTableSQL := `CREATE TABLE IF NOT EXISTS fipe_tables (id INTEGER PRIMARY KEY AUTOINCREMENT, codigo INTEGER, mes TEXT);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/reference-table", func(w http.ResponseWriter, r *http.Request) {
		currentMonth := time.Now().Format("janeiro/2006")

		var codigo int
		var mes string
		err = db.QueryRow("SELECT codigo, mes FROM fipe_tables WHERE mes = ?", currentMonth).Scan(&codigo, &mes)

		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if codigo > 0 {
			json.NewEncoder(w).Encode(codigo)
		}

		if r.Method != "GET" {
			http.Error(w, "Invalid request method.", http.StatusMethodNotAllowed)
			return
		}

		apiUrl := "https://veiculos.fipe.org.br/api/veiculos/ConsultarTabelaDeReferencia"

		request, err := http.NewRequest("POST", apiUrl, nil)
		request.Header.Set("Content-Type", "application/json")

		if err != nil {
			fmt.Printf("Failed to mount request %s\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		client := &http.Client{}
		response, err := client.Do(request)

		if err != nil {
			fmt.Printf("The HTTP request failed with error %s\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		responseBody, err := io.ReadAll(response.Body)

		if err != nil {
			fmt.Printf("Failed to parse response %s\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		var tableReferences []TableReference
		json.Unmarshal(responseBody, &tableReferences)

		var latestTable = tableReferences[0]

		fmt.Printf("Latest : %+v", latestTable.Code)
		json.NewEncoder(w).Encode(latestTable.Code)

		defer response.Body.Close()
	})

	http.ListenAndServe(":8080", nil)
}
