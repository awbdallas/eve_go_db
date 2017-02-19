# eve_go_db
Trying to pull eve market information and populate a db


Small program to pull information about a particular market in EvE online and populate a DB. I currently run it every 3 hours as a cron job. 

Setup
-----

Run it with preset regions
./setup.sh
./control.sh --start

Adding new regions (I'll change this later) requres your to modify eve.go:81 and add the region number

