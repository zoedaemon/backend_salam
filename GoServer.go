// GoServer
package main

import (
	"encoding/json"
	"fmt"
	"net"
	//	"strings"
	"database/sql"

	"github.com/RadhiFadlillah/go-sastrawi"
)

func Server(tags_obj map[string]*Tags) {
	ln, err := net.Listen("tcp", ":1999")
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		c, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleConnection(c, tags_obj)
	}
}

func handleConnection(c net.Conn, tags_obj map[string]*Tags) {
	// we create a decoder that reads directly from the socket
	d := json.NewDecoder(c)

	secret := "2183781237693280uijshadj%%$ds"

	var msg Pelaporan

	err := d.Decode(&msg)

	//if strings.Compare(secret, msg.Secret) == 0 {
	if secret == msg.Secret {
		fmt.Println(msg, err)

		// Pecah kalimat menjadi kata-kata menggunakan tokenizer
		tokenizer := sastrawi.NewTokenizer()
		words := tokenizer.Tokenize(msg.SMS)

		// Ubah kata berimbuhan menjadi kata dasar
		stemmer := sastrawi.NewStemmer(sastrawi.DefaultDictionary)
		for _, word := range words {
			SingleTag, ok := tags_obj[stemmer.Stem(word)]
			if ok {
				fmt.Printf("XXXXXXX %s => %s XXXXXXX\n", word, SingleTag.Root)
			} else {
				fmt.Printf("%s => %s\n", word, stemmer.Stem(word))
			}
		}
	} else {
		fmt.Println("Akses ilegal...!!!")
	}

	c.Close()

}

func main() {
	var TagsObj map[string]*Tags
	var DB *sql.DB

	DB = initDB()
	TagsObj = getTags(DB)

	fmt.Println("Running SALAM Service...")
	Server(TagsObj)
	//defer DB.Close()
}
