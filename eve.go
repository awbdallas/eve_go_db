package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/lib/pq"
)

var ITEM_FILE_PATH = `/var/tmp/eve_db_data/types.json`
var STATION_FILE_PATH = `/var/tmp/eve_db_data/station_types.json`
var REGIONS_TO_WATCH = `/var/tmp/eve_db_data/regions_to_watch`

// Var ERROR_LOG = `/var/tmp/eve_db_data/error.log`

type EveHistoryItem struct {
	OrderCount int     `json:"order_count"`
	LowPrice   float64 `json:"lowest"`
	HighPrice  float64 `json:"highest"`
	AvgPrice   float64 `json:"average"`
	Volume     int     `json:"volume"`
	Date       string  `json:"date"`
}

type EveHistoryRequest struct {
	Items []EveHistoryItem
}

type HistoryRequest struct {
	url      string
	RegionID int
	TypeID   int
	success  bool
	result   EveHistoryRequest
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
	Buy       bool    `json:"is_buy_order"`
	Issued    string  `json:"issued"`
	Price     float64 `json:"price"`
	Volume    int     `json:"volume_remain"`
	Range     string  `json:"range"`
	StationID int     `json:"location_id"`
	TypeID    int     `json:"type_id"`
	Duration  int     `json:"duration"`
}

type EveOrderRequest struct {
	Items []EveOrder
}

func main() {

	var db *sql.DB

	db = InitDB()
	defer db.Close()
	CreateDB(db)
	PopulateItemTable(db)
	PopulateStationTable(db)
	regions := GetRegionsFromFile()

	for {
		for _, region := range regions {
			PopulateOrdersTable(db, region)
			PopulateHistoryTable(db, region)

		}
		time.Sleep(3 * time.Hour)
	}
}

func GetRegionsFromFile() []int {
	// Regions are line seperated and 8 digit numbers
	fh, _ := os.Open(REGIONS_TO_WATCH)
	scanner := bufio.NewScanner(fh)
	var regionids []int

	for scanner.Scan() {
		line := scanner.Text()
		region, _ := strconv.Atoi(line)
		regionids = append(regionids, region)
	}

	return regionids
}

func GetAllRegions(db *sql.DB) []int {
	// Need all of the regions that have a station in them
	// TODO: Probably need a table for just systems instead of just stations
	var regionids []int

	all_region_sql := `
	SELECT DISTINCT regionid FROM stations
	`
	rows, err := db.Query(all_region_sql)
	CheckErr(err)
	defer rows.Close()

	for rows.Next() {
		var regionid int
		err = rows.Scan(&regionid)
		CheckErr(err)
		regionids = append(regionids, regionid)
	}

	return regionids

}

func ClearOrdersTable(db *sql.DB, region_id int) {
	/*
	 * I only want current orders. I don't care about what they're currently at
	 * since I can get any info I want from the history. So, delete all orders and then
	 * Repopulate is the idea here
	 */
	_, err := db.Exec("DELETE FROM  market_orders WHERE regionid = $1", region_id)
	CheckErr(err)
}

func PopulateHistoryTable(db *sql.DB, region_id int) {
	market_items := GetMarketItems(db)
	endpoint := `https://esi.tech.ccp.is/latest/markets/`

	requests := make([]HistoryRequest, len(market_items))

	// Loading struct with url
	for index, item := range market_items {
		var request HistoryRequest
		request.url = (endpoint + strconv.Itoa(region_id) + `/history/?type_id=` +
			strconv.Itoa(item) + `&datasource=tranquility`)
		request.RegionID = region_id
		request.TypeID = item
		requests[index] = request
	}

	// Need the channels for jobs and what's returned
	jobs := make(chan HistoryRequest, len(market_items))
	results := make(chan HistoryRequest, len(market_items))
	// Giving the workers a struct and then we're just going to have them populate it
	for w := 1; w <= 25; w++ {
		go HistoryWorker(w, jobs, results)
	}

	for _, request := range requests {
		jobs <- request
	}
	close(jobs)

	// we only care about requests that were successful, if so we pass them to history
	for i := 1; i <= len(market_items); i++ {
		request := <-results
		if request.success == true {
			StoreEveItemHistory(db, request.result.Items, request.TypeID,
				request.RegionID)
		}
	}
}

func PopulateOrdersTable(db *sql.DB, region_id int) {
	var eveorder EveOrderRequest
	base_url := `https://esi.tech.ccp.is/latest/markets/` + strconv.Itoa(region_id) + `/orders/?order_type=all&`
	curr_page_count := 1
	ClearOrdersTable(db, region_id)

	for {
		holding_url := base_url + `page=` + strconv.Itoa(curr_page_count) + `&datasource=tranquility`
		resp := ReliableGet(holding_url, 5)
		if resp == nil {
			continue
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if string(body) == `[]` {
			break
		}
		err = json.Unmarshal(body, &eveorder.Items)
		if err != nil {
			continue
		}
		StoreEveOrders(db, eveorder.Items, region_id)
		curr_page_count += 1

	}
}

func ReliableGet(url string, tries int) *http.Response {
	timeout := time.Duration(30 * time.Second)

	client := http.Client{
		Timeout: timeout,
	}

	for i := 0; i < tries; i++ {
		resp, err := client.Get(url)

		if err != nil || resp.StatusCode != 200 {
			continue
		} else {
			return resp
		}
	}

	return nil
}

/*
* Grabbed from: https://gobyexample.com/worker-pools
 */
func HistoryWorker(id int, jobs <-chan HistoryRequest, results chan<- HistoryRequest) {
	for job := range jobs {
		results <- ItemHistoryRequest(job)
	}
}

func ItemHistoryRequest(request HistoryRequest) HistoryRequest {
	resp := ReliableGet(request.url, 5)
	if resp == nil {
		request.success = false
		return request
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		request.success = false
		return request
	}
	err = json.Unmarshal(body, &request.result.Items)
	if err != nil {
		request.success = false
		return request
	}
	request.success = true

	return request

}

func StoreEveItemHistory(db *sql.DB, orderhistory []EveHistoryItem,
	typeID int, regionID int) {

	var amount_of_days sql.NullInt64

	err := db.QueryRow(
		"SELECT current_date - max(date) FROM market_data WHERE typeid = $1 AND regionid = $2",
		typeID, regionID).Scan(&amount_of_days)
	CheckErr(err)
	if amount_of_days.Valid {
		days_to_get := len(orderhistory) - int(amount_of_days.Int64)
		if days_to_get >= 0 {
			orderhistory = orderhistory[days_to_get:]
		}
		CheckErr(err)
	}

	txn, err := db.Begin()
	CheckErr(err)

	insert_statement := `
	INSERT INTO market_data (typeid, regionid, ordercount, lowprice, highprice, avgprice, volume, date)
	  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	  ON CONFLICT ON CONSTRAINT unique_typeid_date DO NOTHING;
	`
	stmt, err := txn.Prepare(insert_statement)
	CheckErr(err)

	for _, item := range orderhistory {
		_, err := stmt.Exec(typeID, regionID, item.OrderCount,
			item.LowPrice, item.HighPrice, item.AvgPrice, item.Volume, item.Date)
		CheckErr(err)
	}

	err = stmt.Close()
	CheckErr(err)

	err = txn.Commit()
	CheckErr(err)
}

func StoreEveOrders(db *sql.DB, eveorders []EveOrder, region_id int) {

	txn, err := db.Begin()
	CheckErr(err)

	stmt, err := txn.Prepare(pq.CopyIn("market_orders", "issued", "buy",
		"price", "volume", "stationid", "range", "typeid", "duration", "regionid"))
	CheckErr(err)

	for _, item := range eveorders {
		_, err := stmt.Exec(item.Issued, item.Buy, item.Price, item.Volume,
			item.StationID, item.Range, item.TypeID, item.Duration, region_id)
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
	user := os.Getenv("PGUSER")
	db_name := os.Getenv("PGDATABASE")
	passwd := os.Getenv("PGPASSWORD")

	db, err := sql.Open("postgres", fmt.Sprintf("user=%s dbname=%s password=%s",
		user, db_name, passwd))
	CheckErr(err)
	if db == nil {
		panic("db nil")
	}
	return db
}

func CreateDB(db *sql.DB) {

	item_table := `
	CREATE TABLE IF NOT EXISTS items(
		typeid INT NOT NULL PRIMARY KEY,
		groupid INT,
		typename TEXT,
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
		date DATE,
		CONSTRAINT unique_typeid_date UNIQUE(typeid, date, regionid)
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
	  typeid INT,
	  regionid BIGINT
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

	add_item_no_conflict := `
	INSERT INTO items (typeid, groupid, typename, volume, market)
	  VALUES ($1, $2, $3, $4, $5)
	  ON CONFLICT (typeid) DO NOTHING;
	`

	file, err := ioutil.ReadFile(ITEM_FILE_PATH)
	CheckErr(err)

	var items []EveItem
	json.Unmarshal(file, &items)

	txn, err := db.Begin()
	CheckErr(err)

	stmt, err := txn.Prepare(add_item_no_conflict)
	CheckErr(err)

	for _, item := range items {
		_, err := stmt.Exec(item.TypeID, item.GroupID, item.TypeName,
			item.Volume, item.Market)
		CheckErr(err)
	}

	err = stmt.Close()
	CheckErr(err)

	err = txn.Commit()
	CheckErr(err)
}

func PopulateStationTable(db *sql.DB) {

	add_station_no_conflict := `
	INSERT INTO stations (stationid, regionid, solarsystemid, stationname)
	  VALUES ($1, $2, $3, $4)
	  ON CONFLICT (stationid) DO NOTHING;
	`

	file, err := ioutil.ReadFile(STATION_FILE_PATH)
	CheckErr(err)

	var stations []StationType
	json.Unmarshal(file, &stations)

	txn, err := db.Begin()
	CheckErr(err)

	stmt, err := txn.Prepare(add_station_no_conflict)
	CheckErr(err)
	defer stmt.Close()

	for _, station := range stations {
		_, err := stmt.Exec(station.StationID, station.RegionID,
			station.SolarSystemID, station.StationName)
		CheckErr(err)
	}

	err = stmt.Close()
	CheckErr(err)

	err = txn.Commit()
	CheckErr(err)
}

func CheckErr(err error) {
	// TODO deal with errors better. This is awful. Program needs to run all the time
	if err != nil {
		panic(err)
	}
}
