#!/usr/bin/zsh

# Environment variables
echo "export POSTGRES_DBNAME=eve_market_data" >> ~/.zshrc
echo "export POSTGRES_USER=eve_market_user" >> ~/.zshrc
# You should probs change that if you're using this. 
echo "export POSTGRES_PASSWORD=holding" >> ~/.zshrc

# SETUP POSTGRES
sudo -H -u postgres bash -c "psql -c \"CREATE USER eve_market_user with password 'holding';\""
sudo -H -u postgres bash -c "psql -c \"CREATE DATABASE eve_market_data;\""
sudo -H -u postgres bash -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE eve_market_data to eve_market_user;\""

# Linking files (assuming running from current directory)
ln -s $PWD/data /var/tmp/eve_db_data
