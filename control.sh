#!/usr/bin/zsh

[ $# -lt 1 ] && echo "--stop --start --restart" && exit 1 
PID_FILE=/tmp/eve_db.pid
PROGRAM_FILE=$GOPATH/src/github.com/awbdallas/go_eve_db/eve.go
PROGRAM=$GOBIN/eve
LOG_FILE=/tmp/eve_db_log


while [ $# -gt 0 ]
do 
  case "$1" in 
    --stop) 
      if [ $(pgrep -f "eve" | wc -l) -ne 0 ]; then
        for x in $(pgrep -f "eve"); do
          kill $x
        done
        echo "Script Stopping" >> $LOG_FILE
      else
        echo "Program not found running"
      fi
      ;;
    --start) 
      if [ $(pgrep -f "eve" | wc -l) -eq 0 ]; then
        if [ "$PROGRAM_FILE" -ot "$PROGRAM" ]; then
          go install $PROGRAM_FILE
          echo "Starting Script" >> $LOG_FILE
          $GOBIN/eve >> $LOG_FILE 2>&1 &
        else
          $GOBIN/eve >> $LOG_FILE 2>&1 &
          echo "Starting Script" >> $LOG_FILE
        fi
      else
        echo "Program Still Running. Please Stop with --stop"
      fi
      ;;
    --restart) 
      if [ $(pgrep -f "eve" | wc -l) -eq 0 ]; then
        echo "Script Stopping" >> $LOG_FILE
        for x in $(pgrep -f "eve"); do
          kill $x
        done
      else
        echo "Program not current running. Just starting instead"
      fi
      
      if [ "$PROGRAM_FILE" -ot "$PROGRAM" ]; then
        go install $PROGRAM_FILE
      fi
      echo "Script Starting" >> $LOG_FILE
      $GOBIN/eve >> $LOG_FILE 2>&1 &
      ;;
    --setup)
      echo -n "What will the DBNAME be? : "
      read dbname
      echo -n "What will the User be? : "
      read username
      echo "What will the user password be? :"
      read userpasswd
      # Environment variables
      echo "export POSTGRES_DBNAME=$dbname" >> ~/.zshenv
      echo "export POSTGRES_USER=$username" >> ~/.zshenv
      # You should probs change that if you're using this. 
      echo "export POSTGRES_PASSWORD=$userpasswd" >> ~/.zshenv

      # SETUP POSTGRES
      sudo -H -u postgres bash -c "psql -c \"CREATE USER $username with password '$userpasswd';\""
      sudo -H -u postgres bash -c "psql -c \"CREATE DATABASE $dbname;\""
      sudo -H -u postgres bash -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE $dbname to $username;\""
      
      # Linking files (assuming running from current directory)
      ln -s $PWD/data /var/tmp/eve_db_data
      go install $PWD/eve.go
      ;;
  esac
  shift
done
