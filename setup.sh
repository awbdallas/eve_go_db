#!/usr/bin/zsh

# Environment variables
echo "export POSTGRES_DBNAME=eve_market_data" >> ~/.zshenv
echo "export POSTGRES_USER=eve_market_user" >> ~/.zshenv
# You should probs change that if you're using this. 
echo "export POSTGRES_PASSWORD=holding" >> ~/.zshenv

echo -n > /tmp/eve_db.pid

# SETUP POSTGRES
sudo -H -u postgres bash -c "psql -c \"CREATE USER eve_market_user with password 'holding';\""
sudo -H -u postgres bash -c "psql -c \"CREATE DATABASE eve_market_data;\""
sudo -H -u postgres bash -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE eve_market_data to eve_market_user;\""

# Linking files (assuming running from current directory)
ln -s $PWD/data /var/tmp/eve_db_data

go install $PWD/eve.go
