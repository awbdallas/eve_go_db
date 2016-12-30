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
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var ITEM_FILE_PATH = `./data/types.json`
var STATION_FILE_PATH = `./data/station_types.json`
var DB_PATH = `./data/eve.db`

func main() {
	flag.Parse()

	var db *sql.DB

	if _, err := os.Stat(DB_PATH); os.IsNotExist(err) {
		db = InitDB(DB_PATH)
		defer db.Close()
		CreateDB(db)
		PopulateItemTable(db)
		PopulateStationTable(db)
	} else {
		db = InitDB(DB_PATH)
		defer db.Close()
	}
}

/*
* Purpose: Select all items that are marketable
* Parameters: pointer to DB connection
* Returns: int slice containing all the typeids
 */
func GetMarketItems(db *sql.DB) []int {
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
* Purpose: create db if doesn't exist
* Returns: Nothing
* Parameters: pointer to DB
 */
func CreateDB(db *sql.DB) {

	item_table := `
	CREATE TABLE IF NOT EXISTS items(
		typeID INT NOT NULL PRIMARY KEY,
		groupID INT,
		typeName TEXT,
		volume INT,
		market BOOLEAN
	);
	`

	item_history_table := `
	CREATE TABLE IF NOT EXISTS market_data(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		typeID INT,
		regionID INT,
		orderCount INT,
		lowPrice INT,
		highPrice INT,
		avgPrice INT,
		volume INT,
		date DATE
	);
	`

	station_table := `
	CREATE TABLE IF NOT EXISTS station(
	  stationID INT NOT NULL PRIMARY KEY,
	  regionID INT,
	  solarSystemID INT,
	  stationName TEXT
	);	
	`

	market_orders := `
	CREATE TABLE IF NOT EXISTS market_orders(
	  id INTEGER PRIMARY KEY,
	  issued DATETIME,
	  buy BOOLEARN,
	  price INT,
	  volumeEntered INT,
	  stationID INT,
	  volume INT,
	  range TEXT,
	  duration INT,
	  typeID INT
	);
	`

	_, err := db.Exec(item_table)
	CheckErr(err)
	_, err = db.Exec(item_history_table)
	CheckErr(err)
	_, err = db.Exec(market_orders)
	CheckErr(err)
	_, err = db.Exec(station_table)
	CheckErr(err)

}

/*
* Purpose: Store Items gathered from types.json into the db for the  item
* table
* Returns: Nothing
* Notes: Multiply Volume by 1000 since we were having issues with storing
* decimals
 */
func PopulateItemTable(db *sql.DB) {
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

	file, err := ioutil.ReadFile(ITEM_FILE_PATH)
	CheckErr(err)

	var items []EveItem
	json.Unmarshal(file, &items)

	sql_additem := `
	INSERT OR REPLACE INTO items(
		typeID,
		groupID,
		typeName,
		volume,
		market
	) values(?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_additem)
	CheckErr(err)
	defer stmt.Close()

	for _, item := range items {
		_, err := stmt.Exec(item.TypeID, item.GroupID, item.TypeName,
			item.Volume*1000, item.Market)
		CheckErr(err)
	}
}

func PopulateStationTable(db *sql.DB) {

	type StationType struct {
		StationID     int    `json:"stationID,string"`
		RegionID      int    `json:"regionID,string"`
		SolarSystemID int    `json:"solarSystemID,string"`
		StationName   string `json:"stationName"`
	}

	file, err := ioutil.ReadFile(STATION_FILE_PATH)
	CheckErr(err)

	var stations []StationType
	json.Unmarshal(file, &stations)

	sql_additem := `
	INSERT OR REPLACE INTO station(
	  stationID,
	  regionID,
	  solarSystemID,
	  stationName
	) values(?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_additem)
	CheckErr(err)
	defer stmt.Close()

	for _, station := range stations {
		_, err := stmt.Exec(station.StationID, station.RegionID,
			station.SolarSystemID, station.StationName)
		CheckErr(err)
	}
}

/*
* Purpose: Check err (database errors)
* Parameters: err
 */
func CheckErr(err error) {
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
}
