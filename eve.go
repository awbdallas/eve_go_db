package main

import (
	"database/sql"
	//	"encoding/json"
	//	"fmt"
	_ "github.com/mattn/go-sqlite3"
	// 	"io/ioutil"
	"os"
)

func main() {
	const dbpath = "eve.db"
	const items_file_path = "types.json"

	// Checking for DB before we do anything
	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		db := InitDB(dbpath)
		defer db.Close()
		CreateDB(db)
		//file, err := ioutil.ReadFile(items_file_path)
	} else {
		db := InitDB(dbpath)
		defer db.Close()
	}
}

type EveItem struct {
	Volume   float32
	TypeID   int
	GroupID  int
	Market   bool
	TypeName string
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
		Volume DOUBLE,
		Market BOOLEAN
	);
	`
	market_table := `
	CREATE TABLE IF NOT EXISTS market_data(
		TypeID INT NOT NULL PRIMARY KEY,
		SystemID INT,
		Min_sell DOUBLE,
		Max_buy DOUBLE,
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
			item.Volume, item.Market)
		if err2 != nil {
			panic(err2)
		}
	}
}

func StoreMarketData(db *sql.DB, items []EveItem) {
	sql_additem := `
	INSERT OR REPLACE INTO items(
		Id,
		Name,
		Phone,
		InsertedDatetime
	) values(?, ?, ?, CURRENT_TIMESTAMP)
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
