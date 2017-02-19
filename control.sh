#!/usr/bin/zsh

[ $# -lt 1 ] && echo "--stop --start --restart" && exit 1 
PID_FILE=/tmp/eve_db.pid
PROGRAM_FILE=$GOPATH/src/github.com/awbdallas/go_eve_db/eve.go
LOG_FILE=/tmp/eve_db_log


while [ $# -gt 0 ]
do 
  case "$1" in 
    --stop) 
      cat $PID_FILE | xargs kill 
      echo -n > $PID_FILE
      ;;
    --start) 
      if ["$(stat -c "%Y" $GOBIN/eve)" -lt "$(stat -c "%Y" $PROGRAM_FILE)"]; then
        go install $PROGRAM_FILE
      fi
      $GOBIN/eve & 2>&1 > $LOG_FILE
      pgrep -f "eve" > $PID_FILE
      ;;
    --restart) 
      cat $PID_FILE | xargs kill
      echo -n > $PID_FILE
      if ["$(stat -c "%Y" $GOBIN/eve)" -lt "$(stat -c "%Y" $PROGRAM_FILE)"]; then
        go install $PROGRAM_FILE
      fi
      $GOBIN/eve & 2>&1 > $LOG_FILE
      pgrep -f "eve" > $PID_FILE
      ;;
  esac
  shift
done
