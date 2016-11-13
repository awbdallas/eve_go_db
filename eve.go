package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

// TODO Clean this Namespace, little too much for me in terms of structs

type EveItem struct {
	Volume   float32 `json:"volume"`
	TypeID   int     `json:"typeID"`
	GroupID  int     `json:"groupID"`
	Market   bool    `json:"market"`
	TypeName string  `json:"typeName"`
}

type Market_Items struct {
	Items []Market_Item `xml:"marketstat>type"`
}

type Market_Item struct {
	Id          int `xml:"id,attr"`
	SystemID    int
	Min_sell    float32 `xml:"sell>min"`
	Max_buy     float32 `xml:"buy>max"`
	Volume_sell int     `xml:"sell>volume"`
	Volume_buy  int     `xml:"buy>volume"`
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
	raw_market_info := Get_Market_Info(urls)
	parsed_market_info := Parse_Market_Info(raw_market_info)
	Store_Market_Data(db, parsed_market_info)

}

func Parse_Market_Info(market_info []string) []Market_Item {
	var returning_list []Market_Item

	for i := 0; i < len(market_info); i++ {
		var items Market_Items
		b := []byte(market_info[i])
		xml.Unmarshal(b, &items)
		for _, item := range items.Items {
			returning_list = append(returning_list, item)
		}
	}
	return returning_list
}

func Get_Market_Info(urls []string) []string {
	var market_info []string
	for i := 0; i < len(urls); i++ {
		resp, err := http.Get(urls[i])
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		market_info = append(market_info, string(body))
	}
	return market_info
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

func Store_Market_Data(db *sql.DB, items []Market_Item) {
	sql_additem := `
	INSERT OR REPLACE INTO market_data(
		TypeID, 
		SystemID,
		Min_sell, 
		Max_buy, 
		Volume_sell,
		Volume_buy
	) values(?, ?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_additem)

	if err != nil {
		panic(err)
	}

	defer stmt.Close()

	for _, item := range items {
		_, err2 := stmt.Exec(item.Id, `30000142`, (item.Min_sell * 100),
			(item.Max_buy * 100), item.Volume_sell, item.Volume_buy)

		if err2 != nil {
			panic(err2)
		}
	}
}
