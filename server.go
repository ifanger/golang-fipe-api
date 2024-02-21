package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/goodsign/monday"

	_ "github.com/mattn/go-sqlite3"
)

type TableReference struct {
	Code  int    `json:"Codigo"`
	Month string `json:"Mes"`
}

func main() {
	port := 8080
	currentMonth := monday.Format(time.Now(), "January/2006", monday.LocalePtBR)

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
		var codigo int
		var mes string
		err = db.QueryRow("SELECT codigo, mes FROM fipe_tables WHERE mes = ?", currentMonth).Scan(&codigo, &mes)

		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if codigo > 0 {
			fmt.Printf("Current month %s was found on cache\n", currentMonth)
			json.NewEncoder(w).Encode(codigo)
			return
		}

		fmt.Printf("Current month %s not found on cache, fetching...\n", currentMonth)

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

		latestMonth := formatMonth(tableReferences[0].Month)
		latestCode := tableReferences[0].Code

		fmt.Printf("Inserting into database %d %s\n", latestCode, latestMonth)
		_, err = db.Exec("INSERT INTO fipe_tables (codigo, mes) VALUES (?, ?)", latestCode, latestMonth)

		if err != nil {
			fmt.Printf("Failed to insert into database %s\n", err)
		}

		json.NewEncoder(w).Encode(latestCode)

		defer response.Body.Close()
	})

	http.ListenAndServe(":8080", nil)
}

func formatMonth(month string) string {
	return strings.Trim(strings.ToLower(month), " ")
}
