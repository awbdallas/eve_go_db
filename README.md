# eve_go_db
Trying to pull eve market information and populate a db


Small program to pull information about a particular market in EvE online and populate a DB. I currently run it every 3 hours as a cron job. 

Setup
-----

export POSTGRES_USER="whateveritis"
export POSTGRES_DBNAME="whateveritis"


Example running:
go run eve.go -region regionid_here
