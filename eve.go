package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"os"
	"strconv"
)

type EveItem struct {
	Volume   float32 `json:"volume"`
	TypeID   int     `json:"typeID"`
	GroupID  int     `json:"groupID"`
	Market   bool    `json:"market"`
	TypeName string  `json:"typeName"`
}

func main() {
	const dbpath = "eve.db"
	const items_file_path = "types.json"
	var db *sql.DB
	// Checking for DB before we do anything
	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		db = InitDB(dbpath)
		defer db.Close()
		CreateDB(db)
		file, err := ioutil.ReadFile(items_file_path)

		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		var items []EveItem
		json.Unmarshal(file, &items)
		StoreItem(db, items)
	} else {
		db = InitDB(dbpath)
		defer db.Close()
	}

	urls := Make_URL(db)
	// Needed to use it once
	print(urls[0])
}

func Make_URL(db *sql.DB) []string {
	typeids := Get_Market_Items(db)
	default_system := "30000142"
	base_url := "http://api.eve-central.com/api/marketstat?"
	var urls []string

	url := base_url
	for i := 0; i < len(typeids); i++ {
		// Eve-central limits to 100 items per query
		if (i%100 == 0) && (i != 0) {
			// TODO add CLI or something to figure out what system to grab info
			// for
			url += ("usesystem=" + default_system)
			urls = append(urls, url)
			url = base_url
		} else {
			url += ("typeid=" + strconv.Itoa(typeids[i]) + "&")
		}
	}
	return urls
}

func Get_Market_Items(db *sql.DB) []int {
	// SELECT all items that are on the market
	sql_readall := `
	SELECT TypeID FROM items
	WHERE Market = 1
	`

	rows, err := db.Query(sql_readall)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var result []int
	for rows.Next() {
		var typeid int
		//err2 := rows.Scan(&item.Id, &item.Name, &item.Phone)
		err2 := rows.Scan(&typeid)
		if err2 != nil {
			panic(err2)
		}
		result = append(result, typeid)
	}

	return result
}

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		panic(err)
	}
	if db == nil {
		panic("db nil")
	}
	return db
}

func CreateDB(db *sql.DB) {
	// create table if not exists
	item_table := `
	CREATE TABLE IF NOT EXISTS items(
		TypeID INT NOT NULL PRIMARY KEY,
		GroupID INT,
		TypeName TEXT,
		Volume INT,
		Market BOOLEAN
	);
	`
	market_table := `
	CREATE TABLE IF NOT EXISTS market_data(
		TypeID INT NOT NULL PRIMARY KEY,
		SystemID INT,
		Min_sell INT,
		Max_buy INT,
		Volume_sell INT,
		Volume_buy INT,
		date_of_info date default CURRENT_DATE

	);
	`

	_, err := db.Exec(item_table)

	if err != nil {
		panic(err)
	}
	_, err = db.Exec(market_table)

	if err != nil {
		panic(err)
	}
}

/*
TODO optimize this. Possible do bulk insert
We're storing it as an int because of issues with float,
so we're multiplying by 1000 and we'll just have to keep
the math together
*/
func StoreItem(db *sql.DB, items []EveItem) {
	sql_additem := `
	INSERT OR REPLACE INTO items(
		TypeID,
		GroupID,
		TypeName,
		Volume,
		Market
	) values(?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_additem)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, item := range items {
		_, err2 := stmt.Exec(item.TypeID, item.GroupID, item.TypeName,
			item.Volume*1000, item.Market)
		if err2 != nil {
			panic(err2)
		}
	}
}

/*
func StoreMarketData(db *sql.DB, items []EveItem) {
	sql_additem := `
	INSERT OR REPLACE INTO market_data(
		TypeID,
		SystemID,
		Min_sell,
		Max_buy,
		Volume_sell,
		Volume_buy,
	) values(?, ?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_additem)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, item := range items {
		//_, err2 := stmt.Exec(item.Id, item.Name, item.Phone)
		_, err2 := stmt.Exec(item.TypeID)
		if err2 != nil {
			panic(err2)
		}
	}
}

func ReadItem(db *sql.DB) []EveItem {
	sql_readall := `
	SELECT Id, Name, Phone FROM items
	ORDER BY datetime(InsertedDatetime) DESC
	`

	rows, err := db.Query(sql_readall)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var result []EveItem
	for rows.Next() {
		item := EveItem{}
		//err2 := rows.Scan(&item.Id, &item.Name, &item.Phone)
		err2 := rows.Scan()
		if err2 != nil {
			panic(err2)
		}
		result = append(result, item)
	}
	return result
}
*/
