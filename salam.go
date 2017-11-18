package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	//	"strings"

	"github.com/RadhiFadlillah/go-sastrawi"

	_ "github.com/go-sql-driver/mysql"
)

type Pelaporan struct {
	NoTelp string `json:"no-telp"`
	SMS    string `json:"sms"`
	Secret string `json:"secret"`
}

type PelaporanCleaned struct {
	NoTelp     string
	Pesan      string
	ScoreTotal float64
	IsSpam     bool
	EmbedUrl   string
}

type Tags struct {
	id        int64
	Anchestor string //word root yg asli
	Root      string
	//misal kebakaran-lahan, kebakaran/bakar di map sisanya
	TagsMultiWord []string //column Stemmed
	//rangking prioritas tags
	Score float64
	//single=1, double=2, more=3; TODO: contoh kata gabungan untuk type more apa ya ?
	TypeWord string //byte
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

func getTags(db *sql.DB) map[string]*Tags {

	var NewMapper map[string]*Tags
	NewMapper = make(map[string]*Tags)

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

		var curr_root_word string
		//simpan object Tags baru
		TagsObj := new(Tags)
		// Now do something with the data.
		// Here we just print each column as a string.
		var value string
		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = "NULL"
			} else {
				value = string(col) //TODO: pake interface{} aza dah terlalu mahal convert string
			}
			if columns[i] == "root_word" {
				curr_root_word = value
				TagsObj.Root = value
				TagsObj.Anchestor = value
				fmt.Println(">> ", columns[i], ": ", value)
			} else if columns[i] == "stemmed" {
				fmt.Println(">>-->> ", columns[i], ": ", value)
			} else if columns[i] == "int" {
				TagsObj.id, _ = strconv.ParseInt(value, 10, 64)
				fmt.Println(">>-->> ", columns[i], ": ", value)
			} else if columns[i] == "type_word" {
				TagsObj.TypeWord = value
			} else if columns[i] == "score" {
				TagsObj.Score, _ = strconv.ParseFloat(value, 32)
			} else {
				fmt.Println(columns[i], ": ", value)
			}
		}
		//simpan object Tags baru
		NewMapper[curr_root_word] = TagsObj

		fmt.Println("-----------------------------------")
	}
	if err = rows.Err(); err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	return NewMapper
}

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

		go func(c net.Conn) {
			for {
				message, _, err := bufio.NewReader(c).ReadLine()
				if err == io.EOF {
					fmt.Println("End Of File")
					break
				}
				// sample process for string received
				//newmessage := strings.ToLower(string(message))
				//fmt.Println("Message Received:", newmessage)
				fmt.Println("Message Received:", string(message))
				go handleConnection(message, tags_obj)
			}
			defer c.Close()
		}(c)

		//	d := json.NewDecoder(c)
		//	go handleConnection(c, d, tags_obj)

	}
}

func handleConnection(newmsg []byte, tags_obj map[string]*Tags) {
	// we create a decoder that reads directly from the socket
	//d := json.NewDecoder(c)

	secret := "2183781237693280uijshadj^^^^ds"

	var msg Pelaporan

	err := json.Unmarshal(newmsg, &msg)

	if err != nil {
		log.Fatalln(err.Error())
	}

	//if strings.Compare(secret, msg.Secret) == 0 {
	if secret == msg.Secret {

		//menyimpan score total pelaporan
		var ScoreTotal float64

		fmt.Println(msg, err)
		// Pecah kalimat menjadi kata-kata menggunakan tokenizer
		tokenizer := sastrawi.NewTokenizer()
		words := tokenizer.Tokenize(msg.SMS)

		// Ubah kata berimbuhan menjadi kata dasar
		stemmer := sastrawi.NewStemmer(sastrawi.DefaultDictionary)

		for _, word := range words {
			SingleStemmed := stemmer.Stem(word)
			SingleTag, ok := tags_obj[SingleStemmed]
			if ok {
				fmt.Printf("XXXXXXX %s => %s XXXXXXX\n", word, SingleStemmed)
				ScoreTotal += SingleTag.Score
			} else {
				fmt.Printf("%s => %s\n", word, SingleStemmed)
			}
		}

		fmt.Printf("XXXXXXX ScoreTotal => %f XXXXXXX\n", ScoreTotal)

	} else {
		fmt.Println("Akses ilegal...!!!")
	}

	//defer c.Close()
}
