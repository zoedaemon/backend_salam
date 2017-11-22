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

	DB = initDB("root:@/salamdb")
	TagsObj = getTags(DB)

	fmt.Println("Running SALAM Service...")
	Server(DB, TagsObj)

	//defer DB.Close()
}
