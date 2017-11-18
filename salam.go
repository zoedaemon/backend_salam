package main

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Pelaporan struct {
	NoTelp string `json:"no-telp"`
	SMS    string `json:"sms"`
	Secret string `json:"secret"`
}

type Tags struct {
	Anchestor string //word root yg asli
	Root      string
	//misal kebakaran-lahan, kebakaran/bakar di map sisanya
	TagsMultiWord []string //column Stemmed
	//rangking prioritas tags
	Score float32
	//single=1, double=2, more=3; TODO: contoh kata gabungan untuk type more apa ya ?
	TypeWord byte
}

func initDB() *sql.DB { //db *sql.DB
	fmt.Print("Setting Database...")
	db, err := sql.Open("mysql", "root:@/salamdb")

	if err != nil {
		fmt.Println("FAILED")
		panic(errors.New("error opening db : " + err.Error()))
	} else {
		fmt.Println("OK")
	}

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		fmt.Println("FAILED")
		panic(errors.New("connected but something wrong : " + err.Error()))
	} else {
		fmt.Println("OK")
	}
	return db
}

func getTags(db *sql.DB) map[string]Tags {

	var NewMapper map[string]Tags

	// Execute the query
	rows, err := db.Query("SELECT * FROM tags")
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		// Now do something with the data.
		// Here we just print each column as a string.
		var value string
		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			if columns[i] == "stemmed" {
				fmt.Println(">> ", columns[i], ": ", value)
			} else {
				fmt.Println(columns[i], ": ", value)
			}
		}
		fmt.Println("-----------------------------------")
	}
	if err = rows.Err(); err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	return NewMapper
}
