package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

var ITEM_FILE_PATH = `./data/types.json`
var STATION_FILE_PATH = `./data/station_types.json`
var DB_PATH = `./data/eve.db`

type EveHistoryItem struct {
	OrderCount int     `json:"orderCount"`
	LowPrice   float64 `json:"lowPrice"`
	HighPrice  float64 `json:"highPrice"`
	AvgPrice   float64 `json:"AvgPrice"`
	Volume     int     `json:"volume"`
	Date       string  `json:"date"`
}

type EveHistoryRequest struct {
	Items []EveHistoryItem `json:"items"`
}

type EveItem struct {
	Volume   float64 `json:"volume"`
	TypeID   int     `json:"typeID"`
	GroupID  int     `json:"groupID"`
	Market   bool    `json:"market"`
	TypeName string  `json:"typeName"`
}

type StationType struct {
	StationID     int    `json:"stationID,string"`
	RegionID      int    `json:"regionID,string"`
	SolarSystemID int    `json:"solarSystemID,string"`
	StationName   string `json:"stationName"`
}

type EveOrder struct {
	Buy       bool    `json:"buy"`
	Issued    string  `json:"issued"`
	Price     float64 `json:"price"`
	Volume    int     `json:"volume"`
	Range     string  `json:"range"`
	StationID int     `json:"stationID"`
	TypeID    int     `json:"type"`
	Duration  int     `json:"Duration"`
}

type EveOrderRequest struct {
	Items      []EveOrder `json:"items"`
	TotalCount int        `json:"totalCount"`
	PageCount  int        `json:"pageCount"`
}

func main() {
	var station int
	flag.IntVar(&station, "station", 60003760, "Station ID to pull info for")
	flag.Parse()

	var db *sql.DB

	if _, err := os.Stat(DB_PATH); os.IsNotExist(err) {
		db = InitDB(DB_PATH)
		defer db.Close()
		CreateDB(db)
		PopulateItemTable(db)
		PopulateStationTable(db)
	}

	db = InitDB(DB_PATH)
	PopulateOrdersTable(db, station)
	PopulateHistoryTable(db, station)
	defer db.Close()
}

func PopulateOrdersTable(db *sql.DB, station int) {
	var eveorder EveOrderRequest
	region_id := StationToRegion(db, station)
	url := `https://crest-tq.eveonline.com/market/` + strconv.Itoa(region_id) + `/orders/all/`
	curr_page_count := 1
	total_page_count := 1

	resp, err := http.Get(url)
	CheckErr(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body, &eveorder)
	CheckErr(err)
	total_page_count = eveorder.PageCount
	StoreEveOrders(db, eveorder.Items)
	curr_page_count += 1

	for curr_page_count <= total_page_count {
		resp, err = http.Get(url + `?page=` + strconv.Itoa(curr_page_count))
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		CheckErr(err)
		err = json.Unmarshal(body, &eveorder)
		CheckErr(err)
		StoreEveOrders(db, eveorder.Items)
		curr_page_count += 1
	}
}

func PopulateHistoryTable(db *sql.DB, station int) {
	region_id := StationToRegion(db, station)
	market_items := GetMarketItems(db)
	endpoint := `https://crest-tq.eveonline.com`

	for _, item := range market_items {
		var eveorder EveHistoryRequest
		url := (endpoint + `/market/` + strconv.Itoa(region_id) +
			`/history/?type=` + endpoint + `/inventory/types/` +
			strconv.Itoa(item) + `/`)
		resp, err := http.Get(url)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		CheckErr(err)
		err = json.Unmarshal(body, &eveorder)
		CheckErr(err)
		StoreEveItemHistory(db, eveorder.Items, item, region_id)
	}
}

func StoreEveItemHistory(db *sql.DB, orderhistory []EveHistoryItem,
	typeID int, regionID int) {

	sql_additem := `
	INSERT OR REPLACE into market_data(
	  typeID,
	  regionID,
	  orderCount,
	  lowPrice,
	  highPrice,
	  avgPrice,
	  volume,
	  date
	) values(?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_additem)
	CheckErr(err)
	defer stmt.Close()

	for _, item := range orderhistory {
		_, err := stmt.Exec(typeID, regionID, item.OrderCount,
			item.LowPrice, item.HighPrice, item.AvgPrice, item.Volume, item.Date)
		CheckErr(err)
	}

}

func StoreEveOrders(db *sql.DB, eveorders []EveOrder) {
	sql_additem := `
	INSERT OR REPLACE INTO market_orders(
	  issued,
	  buy,
	  price,
	  volume,
	  range,
	  stationID,
	  typeID,
	  duration
	) values(?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_additem)
	CheckErr(err)
	defer stmt.Close()

	for _, item := range eveorders {
		_, err := stmt.Exec(item.Issued, item.Buy, item.Price, item.Volume,
			item.Range, item.StationID, item.TypeID, item.Duration)
		CheckErr(err)
	}

}

func StationToRegion(db *sql.DB, station int) int {
	sql_get_station_region := `
	SELECT regionID FROM stations
	WHERE stationID = ?;
	`
	var regionID int

	err := db.QueryRow(sql_get_station_region, station).Scan(&regionID)
	CheckErr(err)

	return regionID
}

func GetMarketItems(db *sql.DB) []int {
	// SELECT all items that are on the market
	sql_get_market_items := `
	SELECT TypeID FROM items
	WHERE Market = 1
	`

	rows, err := db.Query(sql_get_market_items)
	CheckErr(err)
	defer rows.Close()

	var result []int
	for rows.Next() {
		var typeid int
		err2 := rows.Scan(&typeid)
		CheckErr(err2)
		result = append(result, typeid)
	}

	return result
}

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	CheckErr(err)
	if db == nil {
		panic("db nil")
	}
	return db
}

func CreateDB(db *sql.DB) {

	item_table := `
	CREATE TABLE IF NOT EXISTS items(
		typeID INT NOT NULL PRIMARY KEY,
		groupID INT,
		typeName TEXT,
		volume REAL,
		market BOOLEAN
	);
	`

	item_history_table := `
	CREATE TABLE IF NOT EXISTS market_data(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		typeID INT,
		regionID INT,
		orderCount INT,
		lowPrice REAL,
		highPrice REAL,
		avgPrice REAL,
		volume INT,
		date DATE
	);
	`

	station_table := `
	CREATE TABLE IF NOT EXISTS stations(
	  stationID INT NOT NULL PRIMARY KEY,
	  regionID INT,
	  solarSystemID INT,
	  stationName TEXT
	);	
	`

	market_orders := `
	CREATE TABLE IF NOT EXISTS market_orders(
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  issued DATETIME,
	  buy BOOLEARN,
	  price REAL,
	  volume INT,
	  stationID INT,
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

func PopulateItemTable(db *sql.DB) {

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
			item.Volume, item.Market)
		CheckErr(err)
	}
}

func PopulateStationTable(db *sql.DB) {

	file, err := ioutil.ReadFile(STATION_FILE_PATH)
	CheckErr(err)

	var stations []StationType
	json.Unmarshal(file, &stations)

	sql_additem := `
	INSERT OR REPLACE INTO stations(
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

func CheckErr(err error) {
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
}
