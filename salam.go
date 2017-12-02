package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
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

	//hehe kelupaan
	TagsOccurence   []*Tags
	LokasiOccurence []*Lokasi
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

type Lokasi struct {
	id         int64
	NamaLokasi string
	Parent     string //Lokasi Parent
	Score      float64
	Timestamp  string
}

var keterangan_tempat = map[string]bool{
	"di":           true,
	"dijln":        true,
	"di jln":       true,
	"di jalan":     true,
	"d":            true,
	"djln":         true,
	"d jln":        true,
	"d jalan":      true,
	"kelurahan":    true,
	"dikelurahan":  true,
	"di kelurahan": true,
	"di klurahan":  true,
}

//hehe nyontek dari https://stackoverflow.com/questions/16551354/how-to-split-a-string-and-assign-it-to-variables-in-golang
type PyString string

//global variable
var chanscore chan *PelaporanCleaned
var wg sync.WaitGroup
var GlobalCounter int
var DataSource string
var secret_conn = "2183781237693280uijshads\n" //secret code tuk koneksi beda di baris baru za XD
var secret = "2183781237693280uijshads"

func initDB(data_source string) *sql.DB { //db *sql.DB
	fmt.Print("Setting Database...")
	//simpan data source k variable global
	DataSource = data_source
	db, err := sql.Open("mysql", data_source)

	if err != nil {
		fmt.Println("FAILED")
		log.Panicln(errors.New("error opening db : " + err.Error()))
	} else {
		fmt.Println("OK")
	}

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		fmt.Println("FAILED")
		log.Panicln(errors.New("connected but something wrong : " + err.Error()))
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
		log.Panicln(err.Error()) // proper error handling instead of panic in your app
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		log.Panicln(err.Error()) // proper error handling instead of panic in your app
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
			log.Panicln(err.Error()) // proper error handling instead of panic in your app
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
		log.Panicln(err.Error()) // proper error handling instead of panic in your app
	}

	return NewMapper
}

func Server(db *sql.DB, tags_obj map[string]*Tags, lokasi_obj map[string]*Lokasi) {
	ln, err := net.Listen("tcp", ":1999")
	if err != nil {
		fmt.Println(err)
		return
	}

	defer func() {
		// recover from panic if one occured. Set err to nil otherwise.
		if recover() != nil {
			err := errors.New("MySQL Error")
			fmt.Println("PANICCCC...", err)
			//mencegah koneksi mati, harus cek pas SELECT query,
			//klo Ping() langsung malah dikira masih aktif, anehhh..
			//..is it go-sql-driver BUG ???
			//TODO: Panggil Pinger dulu jika terjadi panic data
			//	harus tersimpan dulu di temp penyimpanan,
			//	mungkin file tmp kyk C-lang, jd jika tereset bs
			//	ulang proses penyimpanan data dr file tmp
			db = Pinger(db)
		}
	}()

	//var chanreader chan []byte
	defer func() {
		ln.Close()
		fmt.Println("Listener closed...")
	}()

	ctr := counter()

	//proses tiap request yg masuk
	for {
		c, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		//defer c.Close()

		go func(c net.Conn) {

			defer func() {
				// recover from panic if one occured. Set err to nil otherwise.
				if val := recover(); val != nil {
					//err := errors.New("MySQL Error")
					fmt.Println("PANICCCC...", val)
					//mencegah koneksi mati, harus cek pas SELECT query,
					//klo Ping() langsung malah dikira masih aktif, anehhh..
					//..is it go-sql-driver BUG ???
					//TODO: Panggil Pinger dulu jika terjadi panic data
					//	harus tersimpan dulu di temp penyimpanan,
					//	mungkin file tmp kyk C-lang, jd jika tereset bs
					//	ulang proses penyimpanan data dr file tmp
					db = Pinger(db)
				}
			}()
			//untuk keperluan melewatkan sms kyknya gak usah lama2 timeoutnya
			timeoutDuration := 5 * time.Second
			bufreader := bufio.NewReader(c)
			chanscore = make(chan *PelaporanCleaned)
			// Set a deadline for reading. Read operation will fail if no data
			// is received after deadline.
			//c.SetReadDeadline(time.Now().Add(timeoutDuration))

			var messages [][]byte
			var i = 0
			for {
				c.SetReadDeadline(time.Now().Add(timeoutDuration))

				//message, _, err := bufio.NewReader(c).ReadLine()//hahaha bug ini
				message, err := bufreader.ReadBytes('\n')
				if err == io.EOF {
					fmt.Println("EOF")
					break
				}
				//cek koneksi apakah valid dari PHPCLient.php (misalnya)
				if i == 0 && string(message) != secret_conn {
					log.Panic("invalid connection")
					return
				} else {
					i++
					if string(message) != secret_conn {
						fmt.Println("append(messages=", string(message), ")")
						messages = append(messages, message)
					}
				}
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
					handler := handleConnection(message, tags_obj, lokasi_obj)
					if handler == nil {
						return
					}
					chanscore <- handler
				}(msg)
			}

			go func() {

				for valuechan := range chanscore {

					fmt.Println("Proses penyimpanan....valuechan=", valuechan)

					// cek duplikat row
					que := fmt.Sprintf("SELECT id FROM pelaporan WHERE id=%d", valuechan.id)
					var id int
					err := db.QueryRow(que).Scan(&id)
					if err != nil {
						if err.Error() != "sql: no rows in result set" {
							log.Panicln("Query gagal...", err)
						}
					}

					if id > 0 {
						fmt.Println("XXXxx Duplikat entry...!!! ", id)
						continue
					}

					//TODO: buat operasi INSERT lebih sederhana
					stmt, err := db.Prepare("INSERT INTO pelaporan(id, no_telp, pesan, score_total, is_spam, embed_url) " +
						"VALUES(?, ?, ?, ?, ?, ?)")
					if err != nil {
						log.Panicln(err)
					}
					//demi nama keamanan...
					PesanSecured := template.HTMLEscapeString(valuechan.Pesan)
					_, err = stmt.Exec(valuechan.id, valuechan.NoTelp, PesanSecured, valuechan.ScoreTotal, valuechan.IsSpam, valuechan.EmbedUrl)
					if err != nil {
						log.Panicln(err)
					}
					fmt.Println("Simpan Data Berhasil : valuechan=", valuechan)
					fmt.Println("----- : AllTags=", valuechan.TagsOccurence)
					for _, tag := range valuechan.TagsOccurence {
						fmt.Println("----->>>> ", tag.id, " : Tag=", tag.Root, "; Ancestor=", tag.Root)

						que := fmt.Sprintf("SELECT id FROM pelaporan_tags WHERE id_pelaporan=%d"+
							" AND id_tags=%d", valuechan.id, tag.id)
						var id int
						err := db.QueryRow(que).Scan(&id)
						if err != nil {
							if err.Error() != "sql: no rows in result set" {
								log.Panicln("Query gagal...", err)
							}
						}
						//lewatkan aza klo data dah ada
						if id > 0 {
							fmt.Println("XXXxx Duplikat tags...!!! ", id)
							continue
						}

						stmt, err := db.Prepare("INSERT INTO pelaporan_tags(id_pelaporan, id_tags) " +
							"VALUES(?, ?)")
						if err != nil {
							log.Panicln(err)
						}
						_, err = stmt.Exec(valuechan.id, tag.id)
						if err != nil {
							log.Panicln(err)
						}
						fmt.Println("Simpan Tag Berhasil...")
					}

					for _, lokasi := range valuechan.LokasiOccurence {
						fmt.Println("-----ZZZZZ ", lokasi.id, " : Lokasi=", lokasi.NamaLokasi, "; ")
						stmt, err := db.Prepare("INSERT INTO pelaporan_lokasi(id_pelaporan, id_lokasi) " +
							"VALUES(?, ?)")
						if err != nil {
							log.Panicln(err)
						}
						_, err = stmt.Exec(valuechan.id, lokasi.id)
						if err != nil {
							log.Panicln(err)
						}
						fmt.Println("Simpan Lokasi Berhasil...")
					}
					ctr()
				}
				//defer c.Close()
				// Close connection when this function ends
				defer func() {
					fmt.Println("ZZZZzzzzzzz Closing connection...")
					c.Close() //TODO: belum ditest neh
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

func handleConnection(newmsg []byte, tags_obj map[string]*Tags, lokasi_obj map[string]*Lokasi) *PelaporanCleaned {
	// we create a decoder that reads directly from the socket
	//d := json.NewDecoder(c)
	var returnval *PelaporanCleaned

	var msg Pelaporan

	defer func() {
		// recover from panic if one occured. Set err to nil otherwise.
		if recover() != nil {
			err := errors.New("handleConnection Error : " + string(newmsg))
			fmt.Println("PANICCCC...", err)
			//mencegah koneksi mati, harus cek pas SELECT query,
			//klo Ping() langsung malah dikira masih aktif, anehhh..
			//..is it go-sql-driver BUG ???
			//TODO: Panggil Pinger dulu jika terjadi panic data
			//	harus tersimpan dulu di temp penyimpanan,
			//	mungkin file tmp kyk C-lang, jd jika tereset bs
			//	ulang proses penyimpanan data dr file tmp
			//db = Pinger(db)
			return
		}
	}()

	err := json.Unmarshal(newmsg, &msg)

	if err != nil {
		log.Panicln(err.Error())
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

		var TagsOccurence []*Tags
		var LokasiOccurence []*Lokasi
		skipper := 0
		last_word := ""
		last_word_main := ""
		check_word_main := ""
		for _, word := range words {
			if keterangan_tempat[word] {
				fmt.Println("Cek lokasi 1...")
				last_word = word
				skipper = 1
				continue
			}
			if skipper > 0 {
				fmt.Println("Cek lokasi 2...")
				//cek apakah keterangan tempat non tunggal
				if keterangan_tempat[last_word+" "+word] {
					fmt.Println("Cek lokasi 3...")
					last_word = "" //kosongkan untuk pemprosesan nama kelurahan
					skipper = 1
					continue
				} else if skipper < 2 {
					last_word = "" //reset last_word, dgn asumsi keterangan t4nya tunggal
				}
				word = last_word + " " + word
				//TODO: tolower
				SingleLokasi, ok := lokasi_obj[strings.TrimSpace(word)]
				if ok {
					fmt.Printf("########## %s => %s #########\n", word, SingleLokasi.NamaLokasi)
					skipper = 0 //jgn cek lokasi lg kata selanjutnya
				} else {
					//asumsi nama kelurahan hanya 2 kata saja, jgn paksa menemukan t4 yg valid
					if skipper >= 3 {
						skipper = 0
						goto CEKSTEM //jump to CEKSTEM tuk melanjutkan pemprosesan tags
					}
					last_word = last_word + " " + strings.TrimSpace(word)
					skipper++
					fmt.Println(last_word, ", ", skipper)
					continue
				}

				skipper--
				continue
			}
		CEKSTEM:
			SingleStemmed := stemmer.Stem(word)
			SingleTag, ok := tags_obj[SingleStemmed]
			if ok {
				fmt.Printf("XXXXXXX %s => %s XXXXXXX\n", word, SingleStemmed)
				ScoreTotal += SingleTag.Score
				TagsOccurence = append(TagsOccurence, SingleTag)
			} else {
				fmt.Printf("%s => %s\n", word, SingleStemmed)
			}

			//antisipasi kata t4 tidak memiliki kata penghubung di keterangan_tempat
			check_word_main = last_word_main + " " + strings.TrimSpace(word)
			//cek 1 kata t4
			if last_word_main != "" {
				SingleLokasi, ok := lokasi_obj[strings.TrimSpace(last_word_main)]
				if ok {
					fmt.Printf("########## %s => %s #########\n", word, SingleLokasi.NamaLokasi)
					LokasiOccurence = append(LokasiOccurence, SingleLokasi)
				}
			}
			//cek 1 kata t4
			last_word_main = word
			SingleLokasi, ok := lokasi_obj[strings.TrimSpace(check_word_main)]
			if ok {
				fmt.Printf("########## %s => %s #########\n", word, SingleLokasi.NamaLokasi)
				LokasiOccurence = append(LokasiOccurence, SingleLokasi)
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
		returnval = &PelaporanCleaned{msgid, msg.NoTelp, msg.SMS, ScoreTotal, false, "", TagsOccurence, LokasiOccurence}
	} else {
		fmt.Println("Akses ilegal...!!!") //TODO: kok gak muncul
		log.Panicln("Akses ilegal...!!!")
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

func Pinger(db *sql.DB) *sql.DB {
	// Reconnect TODO : bisa sederhanakan dengan cek langsung lewat for loop
	err := db.Ping()
	if err != nil {
		for {
			fmt.Println("FAILED : try to reconnect...")
			db.Close()
			var err error
			db, err = sql.Open("mysql", DataSource)
			//ping ulang
			err = db.Ping()
			if err != nil {
				continue
			} else {
				break //reconnect berhasil
			}
			time.Sleep(time.Second * 1)
		}
	} else {
		fmt.Println("Connection OK")
	}

	return db
}

func getLokasi(db *sql.DB) map[string]*Lokasi {
	var LokasiObj *Lokasi
	var NewMapper map[string]*Lokasi
	NewMapper = make(map[string]*Lokasi)

	//TODO: hapus stemmer di query ini jika yg dari PHP udah slese
	//stemmer := sastrawi.NewStemmer(sastrawi.DefaultDictionary)

	// Execute the query
	rows, err := db.Query("SELECT * FROM lokasi")
	if err != nil {
		log.Panicln(err.Error()) // proper error handling instead of panic in your app
	}

	// Fetch rows
	for rows.Next() {
		LokasiObj = new(Lokasi)
		//simpan object Tags baru
		//		TagsObj := new(Tags)

		// get RawBytes from data
		err = rows.Scan(&LokasiObj.id, &LokasiObj.NamaLokasi, &LokasiObj.Parent, &LokasiObj.Score, &LokasiObj.Timestamp)
		if err != nil {
			log.Panicln(err.Error()) // proper error handling instead of panic in your app
		}

		fmt.Println(LokasiObj)
		NewMapper[strings.ToLower(LokasiObj.NamaLokasi)] = LokasiObj
		fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
	}
	//fmt.Println(NewMapper)
	if err = rows.Err(); err != nil {
		log.Panicln(err.Error()) // proper error handling instead of panic in your app
	}

	return NewMapper
}
