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
	"strings"
	"sync"
	//	"time"

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
	TagsMultiWord []string //column Stemmed TODO tambah score tuk multi stem
	//rangking prioritas tags
	Score float64
	//single=1, double=2, more=3; TODO: contoh kata gabungan untuk type more apa ya ?
	TypeWord string //byte
}

//untuk perulangan pengambilan data-data di kolom stemmed
type MultiStemHelper struct {
	stem string //stem saat ini
	next int    // > 1 untuk pengambilan stem yang akan disimpan di TagsMultiWord
}

//hehe nyontek dari https://stackoverflow.com/questions/16551354/how-to-split-a-string-and-assign-it-to-variables-in-golang
type PyString string

//global variable
var chanscore chan float64
var wg sync.WaitGroup
var GlobalCounter int

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
				fmt.Println(">>-STEMMED->> ", value)
				var Stemmed PyString
				Stemmed = PyString(value)

				ArrStem, err := Stemmed.Split(",")
				if err != nil {
					fmt.Println(">>-ERROR->> ", err)
					continue
				}
				fmt.Println(">>-->> ", columns[i], ": ")

				//menyimpan semua kombinasi string root_word yang mungkin muncul
				var stemmeds []*MultiStemHelper

				//TODO: tidak perlu stemming disini karena php sudah melakukannya
				//		ketika input data ke kolom optional_combination; kolom
				//		stemmed adalah hasil stemmingnya (readonly di formnya nanti)
				for _, Stem := range ArrStem {
					Stem = strings.TrimSpace(Stem)
					if strings.ContainsAny(Stem, "-") {
						ArrSubStem := strings.Split(Stem, "-")
						size_ArrSubStem := len(ArrSubStem)
						iterator := 0
						for _, Stem2orMore := range ArrSubStem {
							var MStem *MultiStemHelper
							if iterator > 0 {
								MStem = &MultiStemHelper{Stem2orMore, 1}
								iterator++
							} else {
								MStem = &MultiStemHelper{Stem2orMore, size_ArrSubStem}
								iterator++
							}
							stemmeds = append(stemmeds, MStem)
							fmt.Println(iterator, "--- 2 > ---", Stem2orMore)
						}
					} else {
						MStem := &MultiStemHelper{Stem, 1}
						stemmeds = append(stemmeds, MStem)
						fmt.Println("--- 1 ---", Stem)
					}

				}
			} else if columns[i] == "id" {
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

	//var chanreader chan []byte
	defer func() {
		ln.Close()
		fmt.Println("Listener closed...")
	}()

	ctr := counter()

	for {
		c, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		//defer c.Close()

		go func(c net.Conn) {

			//untuk keperluan melewatkan sms kyknya gak usah lama2 timeoutnya
			timeoutDuration := 5 * time.Second
			bufreader := bufio.NewReader(c)
			chanscore = make(chan float64)
			// Set a deadline for reading. Read operation will fail if no data
			// is received after deadline.
			//c.SetReadDeadline(time.Now().Add(timeoutDuration))

			var messages [][]byte
			for {
				//c.SetReadDeadline(time.Now().Add(timeoutDuration))

				//message, _, err := bufio.NewReader(c).ReadLine()//hahaha bug ini
				message, err := bufreader.ReadBytes('\n')
				if err == io.EOF {
					fmt.Println("EOF")
					break
				}
				fmt.Println("append(messages=", string(message), ")")
				messages = append(messages, message)

			}

			wg.Add(len(messages))

			for _, msg := range messages {
				fmt.Println("msg:", string(msg))
				go func(message []byte) {
					defer wg.Done()
					// sample process for string received
					//newmessage := strings.ToLower(string(message))
					//fmt.Println("Message Received:", newmessage)
					fmt.Println("Message Received:", string(message))
					chanscore <- handleConnection(message, tags_obj)
				}(msg)
			}

			go func() {

				for valuechan := range chanscore {
					fmt.Println("valuechan=", valuechan)
					ctr()
				}
				//defer c.Close()
				// Close connection when this function ends
				defer func() {
					fmt.Println("ZZZZzzzzzzz Closing connection...")
					//c.Close()
				}()

			}()
			//close(chanscore)

			//go func() {
			wg.Wait()
			close(chanscore)
			//}()
			//time.Sleep(2 * time.Second)
			//wg.Wait()
		}(c)

		//	d := json.NewDecoder(c)
		//	go handleConnection(c, d, tags_obj)

	}
}

func handleConnection(newmsg []byte, tags_obj map[string]*Tags) float64 {
	// we create a decoder that reads directly from the socket
	//d := json.NewDecoder(c)
	var returnval float64

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

		fmt.Printf("-------->>> ScoreTotal => %f <<<--------\n", ScoreTotal)
		returnval = ScoreTotal
	} else {
		fmt.Println("Akses ilegal...!!!")
	}

	return returnval
	//defer c.Close()
}

//static counter
func counter() (f func()) {
	f = func() {
		GlobalCounter++
		fmt.Println("COUNTER=", GlobalCounter)
	}
	return f
}

func (py PyString) Split(str string) ([]string, error) {
	s := strings.Split(string(py), str)
	if len(s) < 2 {
		return s, errors.New("Minimum match not found")
	}
	return s, nil
}
