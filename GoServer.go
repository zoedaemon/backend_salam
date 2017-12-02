// GoServer
package main

import (
	"database/sql"
	//	"errors"
	"fmt"
	//	"strings"
)

func main() {
	var TagsObj map[string]*Tags
	var DB *sql.DB

	DB = initDB("phpmyadmin:adm19adm89@/salamdb")
	TagsObj = getTags(DB)
	LokasiObj := getLokasi(DB)

	fmt.Println("Running SALAM Service...")
	Server(DB, TagsObj, LokasiObj)

	//defer DB.Close()
}
