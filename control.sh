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
  esac
  shift
done
