package main

/*
 * Author: awbdallas
 * Purpose: The purpose is to scrape eve_central for info to a local sqlite db
 * so that I can hopefully track trends in items on eve with the information.
 *
 * Note: I'm not sure how to credit like other people whos code I used for this
 * but this is where I got most of my original code about sqlite and go:
 * https://siongui.github.io/2016/01/09/go-sqlite-example-basic-usage/
 *
 * Possible Ideas for the future for more data:
 * Trying to get more data by using CREST instead of this.
 */

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

/* The EveItem struct is intended to store items that are read in from
 * types.json that should be found in the same folder. This info is primarily
 * to store info into the database into the items table
 */
type EveItem struct {
	Volume   float32 `json:"volume"`
	TypeID   int     `json:"typeID"`
	GroupID  int     `json:"groupID"`
	Market   bool    `json:"market"`
	TypeName string  `json:"typeName"`
}

/* The Market_Items and Market_Item structs are to be used in order to hold
 * the information that we grab from our rest request that return xml. This info
 * is then stored into the database as market_info
 */
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
	// TODO Deal with arguments with options just to make it friendly
	dbpath := os.Args[1]
	items_file_path := os.Args[2]

	var db *sql.DB

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

/*
* Purpose: Parse Market Information with the use of xml library
* Parameters: info comes in as a slice of strings
* Returns: slice of Market_Item
 */
func Parse_Market_Info(raw_market_info []string) []Market_Item {
	var parsed_items []Market_Item

	for _, raw_items := range raw_market_info {
		var items Market_Items
		xml.Unmarshal([]byte(raw_items), &items)
		for _, item := range items.Items {
			parsed_items = append(parsed_items, item)
		}
	}
	return parsed_items
}

/*
* Purpose: Get market info by REST calls
* Parameters: slice of strings that contain urls
* Returns: slice of strings that contain all the info that were getting
* by the rest calls
 */
func Get_Market_Info(urls []string) []string {
	var market_info []string

	for _, url := range urls {
		resp, err := http.Get(url)
		defer resp.Body.Close()
		if err != nil {
			panic(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		market_info = append(market_info, string(body))
	}

	return market_info
}

/*
* Purpose: Make the URLS for querying
* Parameters: db pointer
* Returns: slice of strings that contain the urls
 */
func Make_URL(db *sql.DB) []string {
	typeids := Get_Market_Items(db)
	default_system := "30000263"
	base_url := "http://api.eve-central.com/api/marketstat?"
	var urls []string

	url := base_url
	for i, item := range typeids {
		// Queries limited to 100 at a time
		if (i%100 == 0) && (i != 0) {
			url += ("usesystem=" + default_system)
			urls = append(urls, url)
			url = base_url
		} else {
			url += ("typeid=" + strconv.Itoa(item) + "&")
		}
	}
	return urls
}

/*
* Purpose: Select all items that are marketable
* Parameters: pointer to DB connection
* Returns: int slice containing all the typeids
 */
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
		err2 := rows.Scan(&typeid)
		if err2 != nil {
			panic(err2)
		}
		result = append(result, typeid)
	}

	return result
}

/*
* Purpose: Inital connection to DB
* table
* Returns: Pointer to DB connection
* Parameters: path to file
 */
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

/*
* Purpose: Create two tables for the DB. items and market_data
* table
* Returns: Nothing
* Parameters: pointer to DB
 */
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
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		TypeID INT,
		SystemID INT,
		Min_sell INT,
		Max_buy INT,
		Volume_sell INT,
		Volume_buy INT,
		date_of_info TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
* Purpose: Store Items gathered from types.json into the db for the  item
* table
* Returns: Nothing
* Parameters: pointer to DB and slice of EveItem struct
* Notes: Multiply Volume by 1000 since we were having issues with storing
* decimals
* TODO: Optimize Insert
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
* Purpose: Take in a slice of Market_Item and store them in the database
* Retunrs: Nothing
* Parameters: pointer to DB and slice of struct Market_Item
 */
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
		_, err2 := stmt.Exec(item.Id, `30000263`, (item.Min_sell * 100),
			(item.Max_buy * 100), item.Volume_sell, item.Volume_buy)

		if err2 != nil {
			panic(err2)
		}
	}
}
