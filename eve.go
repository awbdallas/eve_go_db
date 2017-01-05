package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/lib/pq"
)

var ITEM_FILE_PATH = `./data/types.json`
var STATION_FILE_PATH = `./data/station_types.json`

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
	var region int
	flag.IntVar(&region, "region", 10000002, "Region to Pull info for")
	flag.Parse()

	var db *sql.DB

	db = InitDB()
	CreateDB(db)
	PopulateItemTable(db)
	PopulateStationTable(db)
	PopulateOrdersTable(db, region)
	PopulateHistoryTable(db, region)
	defer db.Close()
}

func PopulateOrdersTable(db *sql.DB, region_id int) {
	var eveorder EveOrderRequest
	url := `https://crest-tq.eveonline.com/market/` + strconv.Itoa(region_id) + `/orders/all/`
	curr_page_count := 1
	total_page_count := 1

	ClearOrdersTable(db)

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

func ClearOrdersTable(db *sql.DB) {
	_, err := db.Exec("TRUNCATE market_orders;")
	CheckErr(err)
}

func ClearHistoryTable(db *sql.DB) {
	_, err := db.Exec("TRUNCATE market_data;")
	CheckErr(err)
}

func PopulateHistoryTable(db *sql.DB, region_id int) {
	market_items := GetMarketItems(db)
	endpoint := `https://crest-tq.eveonline.com`

	ClearHistoryTable(db)

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

	txn, err := db.Begin()
	CheckErr(err)

	stmt, err := txn.Prepare(pq.CopyIn("market_data", "typeid", "regionid",
		"ordercount", "lowprice", "highprice", "avgprice", "volume", "date"))
	CheckErr(err)

	for _, item := range orderhistory {
		_, err := stmt.Exec(typeID, regionID, item.OrderCount,
			item.LowPrice, item.HighPrice, item.AvgPrice, item.Volume, item.Date)
		CheckErr(err)
	}

	_, err = stmt.Exec()
	CheckErr(err)

	err = stmt.Close()
	CheckErr(err)

	err = txn.Commit()
	CheckErr(err)

}

func StoreEveOrders(db *sql.DB, eveorders []EveOrder) {

	txn, err := db.Begin()
	CheckErr(err)

	stmt, err := txn.Prepare(pq.CopyIn("market_orders", "issued", "buy",
		"price", "volume", "stationid", "range", "duration", "typeid"))
	CheckErr(err)

	for _, item := range eveorders {
		_, err := stmt.Exec(item.Issued, item.Buy, item.Price, item.Volume,
			item.StationID, item.Range, item.TypeID, item.Duration)
		CheckErr(err)
	}

	_, err = stmt.Exec()
	CheckErr(err)

	err = stmt.Close()
	CheckErr(err)

	err = txn.Commit()
	CheckErr(err)

}

func StationToRegion(db *sql.DB, station int) int {
	var regionID int

	err := db.QueryRow("SELECT regionid FROM stations WHERE stationid = $1",
		station).Scan(&regionID)
	CheckErr(err)

	return regionID
}

func GetMarketItems(db *sql.DB) []int {
	// SELECT all items that are on the market
	sql_get_market_items := `
	SELECT TypeID FROM items
	WHERE Market = 'True'
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

func InitDB() *sql.DB {
	db, err := sql.Open("postgres", "user=awbriggs dbname=eve_market_data")
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
		id SERIAL PRIMARY KEY,
		typeid INT,
		regionid INT,
		ordercount INT,
		lowprice REAL,
		highprice REAL,
		avgprice REAL,
		volume BIGINT,
		date DATE
	);
	`

	station_table := `
	CREATE TABLE IF NOT EXISTS stations(
	  stationid INT NOT NULL PRIMARY KEY,
	  regionid INT,
	  solarsystemid INT,
	  stationname TEXT
	);	
	`

	market_orders := `
	CREATE TABLE IF NOT EXISTS market_orders(
	  id SERIAL PRIMARY KEY,
	  issued TIMESTAMP,
	  buy BOOLEAN,
	  price REAL,
	  volume BIGINT,
	  stationid BIGINT,
	  range TEXT,
	  duration INT,
	  typeid INT
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

	txn, err := db.Begin()
	CheckErr(err)

	stmt, err := txn.Prepare(pq.CopyIn("items", "typeid", "groupid", "typename",
		"volume", "market"))
	CheckErr(err)

	for _, item := range items {
		_, err := stmt.Exec(item.TypeID, item.GroupID, item.TypeName,
			item.Volume, item.Market)
		CheckErr(err)
	}

	_, err = stmt.Exec()
	CheckErr(err)

	err = stmt.Close()
	CheckErr(err)

	err = txn.Commit()
	CheckErr(err)
}

func PopulateStationTable(db *sql.DB) {

	file, err := ioutil.ReadFile(STATION_FILE_PATH)
	CheckErr(err)

	var stations []StationType
	json.Unmarshal(file, &stations)

	txn, err := db.Begin()
	CheckErr(err)

	stmt, err := txn.Prepare(pq.CopyIn("stations", "stationid", "regionid",
		"solarsystemid", "stationname"))
	CheckErr(err)
	defer stmt.Close()

	for _, station := range stations {
		_, err := stmt.Exec(station.StationID, station.RegionID,
			station.SolarSystemID, station.StationName)
		CheckErr(err)
	}

	_, err = stmt.Exec()
	CheckErr(err)

	err = stmt.Close()
	CheckErr(err)

	err = txn.Commit()
	CheckErr(err)
}

func CheckErr(err error) {
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
}
