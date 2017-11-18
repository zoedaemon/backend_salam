// GoServer
package main

import (
	"database/sql"
	"fmt"
	//	"strings"
)

func main() {
	var TagsObj map[string]*Tags
	var DB *sql.DB

	DB = initDB()
	TagsObj = getTags(DB)

	fmt.Println("Running SALAM Service...")
	Server(TagsObj)
	//defer DB.Close()
}
