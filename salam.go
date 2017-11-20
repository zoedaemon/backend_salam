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
	"time"

	"github.com/RadhiFadlillah/go-sastrawi"

	_ "github.com/go-sql-driver/mysql"
)

type Pelaporan struct {
	ID     interface{} `json:"id"` //,omitempty //BUG GOLANG : ID ubah jd id akan terjadi error
	NoTelp string      `json:"no-telp"`
	SMS    string      `json:"sms"`
	Secret string      `json:"secret"`
}

type PelaporanCleaned struct {
	id         int
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
var chanscore chan *PelaporanCleaned
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

	//TODO: hapus stemmer di query ini jika yg dari PHP udah slese
	stemmer := sastrawi.NewStemmer(sastrawi.DefaultDictionary)

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		//simpan object Tags baru
		TagsObj := new(Tags)

		//menyimpan semua kombinasi string root_word yang mungkin muncul
		var stemmeds []*MultiStemHelper

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
				val := stemmer.Stem(value)
				TagsObj.Root = val
				TagsObj.Anchestor = val
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
								MStem = &MultiStemHelper{stemmer.Stem(Stem2orMore), 1}
								iterator++
							} else {
								MStem = &MultiStemHelper{stemmer.Stem(Stem2orMore), size_ArrSubStem}
								iterator++
							}
							stemmeds = append(stemmeds, MStem)
							fmt.Println(iterator, "--- 2 > ---", Stem2orMore)
						}
					} else {
						MStem := &MultiStemHelper{stemmer.Stem(Stem), 1}
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
		//simpan object Tags baru khusus untuk stem word utama
		NewMapper[TagsObj.Root] = TagsObj

		//pemprosesan stem opsional yang sudah diproses php (kolom stemmed)
		for _, stem := range stemmeds {
			TagsObjStem := new(Tags)
			TagsObjStem.id = TagsObj.id
			TagsObjStem.Anchestor = TagsObj.Anchestor
			TagsObjStem.Score = TagsObj.Score
			TagsObjStem.Root = stem.stem

			NewMapper[stem.stem] = TagsObjStem
		}

		fmt.Println("-----------------------------------")
	}
	if err = rows.Err(); err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	return NewMapper
}

func Server(db *sql.DB, tags_obj map[string]*Tags) {
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
			chanscore = make(chan *PelaporanCleaned)
			// Set a deadline for reading. Read operation will fail if no data
			// is received after deadline.
			//c.SetReadDeadline(time.Now().Add(timeoutDuration))

			var messages [][]byte
			for {
				c.SetReadDeadline(time.Now().Add(timeoutDuration))

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

					defer func() {
						// recover from panic if one occured. Set err to nil otherwise.
						if recover() != nil {
							err := errors.New("array index out of bounds")
							fmt.Println("PANICCCC...", err)
						}
					}()

					fmt.Println("Proses penyimpanan....valuechan=", valuechan)

					// cek duplikat row
					que := fmt.Sprintf("SELECT id FROM pelaporan WHERE id=%d", valuechan.id)
					var id float64
					err := db.QueryRow(que).Scan(&id)
					if err != nil {
						fmt.Println("Query gagal...", err)
					}

					if id > 0 {
						fmt.Println("XXXxx Duplikat entry...!!! ", id)
						continue
					}

					stmt, err := db.Prepare("INSERT INTO pelaporan(id, no_telp, pesan, score_total, is_spam, embed_url) " +
						"VALUES(?, ?, ?, ?, ?, ?)")
					if err != nil {
						log.Fatal(err)
					}
					_, err = stmt.Exec(valuechan.id, valuechan.NoTelp, valuechan.Pesan, valuechan.ScoreTotal, valuechan.IsSpam, valuechan.EmbedUrl)
					if err != nil {
						log.Fatal(err)
					}
					fmt.Println("Simpan Data Berhasil : valuechan=", valuechan)
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

func handleConnection(newmsg []byte, tags_obj map[string]*Tags) *PelaporanCleaned {
	// we create a decoder that reads directly from the socket
	//d := json.NewDecoder(c)
	var returnval *PelaporanCleaned

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

		fmt.Printf("-------->>> ScoreTotal => %f <<<-------- %d\n", ScoreTotal, msg.ID)

		//json to uint converter
		//jika msg.id bertipe interface{}
		var n int
		if i, ok := msg.ID.(float64); ok { // yeah, JSON numbers are floats, gotcha!
			n = int(i)
		} else if s, ok := msg.ID.(string); ok {
			ni, err := strconv.Atoi(s[1 : len(s)-1])
			n = int(ni)
			if err != nil {
				fmt.Println("FAILED konversi uint")
			}
		}
		msgid := n
		returnval = &PelaporanCleaned{msgid, msg.NoTelp, msg.SMS, ScoreTotal, false, ""}
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
