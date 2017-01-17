#!/usr/bin/zsh

[ $# -lt 1 ] && echo "--stop --start --restart" && exit 1 
PID_FILE=/tmp/eve_db.pid

while [ $# -gt 0 ]
do 
  case "$1" in 
    --stop) 
      cat $PID_FILE | xargs kill 
      echo -n > $PID_FILE
      ;;
    --start) 
      $GOBIN/eve &
      pgrep -f "eve" > $PID_FILE
      ;;
    --restart) 
      cat $PID_FILE | xargs kill
      echo -n > $PID_FILE
      $GOBIN/eve &
      pgrep -f "eve" > $PID_FILE
      ;;
  esac
  shift
done
