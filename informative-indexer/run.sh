#!/bin/bash

set -e

SCRIPT_DIR=$(cd $(dirname $0) ; pwd -P)

TASK=$1
ARGS=${@:2}

help__run="run <..args> : run indexer"
task__run() {
  local chain=$1

  if [ -z "$chain" ]; then
    echo "usage: $0 run <chain> <id>"
    exit
  fi

  go build -o informative-indexer.bin .

  source .env

  ./informative-indexer.bin run --bootstrap-server $KAFKA_BOOTSTRAP_SERVER \
    --block-results-topic ${chain}-local-informative-indexer-block-results-messages \
    --kafka-api-key $KAFKA_API_KEY \
    --kafka-api-secret $KAFKA_API_SECRET \
    --block-results-consumer-group ${chain}-local-informative-indexer-flusher \
    --block-results-claim-check-bucket ${chain}-local-informative-indexer-large-block-results \
    --claim-check-threshold-mb 1 \
    --db $DB_CONNECTION_STRING \
    --chain $chain \
    --id 1
}

list_all_helps() {
  compgen -v | egrep "^help__.*"
}

NEW_LINE=$'\n'
if type -t "task__$TASK" &>/dev/null; then
  task__$TASK $ARGS
else
  echo "usage: $0 <task> [<..args>]"
  echo "task:"

  HELPS=""
  for help in $(list_all_helps)
  do

    HELPS="$HELPS    ${help/help__/} |-- ${!help}$NEW_LINE"
  done

  echo "$HELPS" | column -t -s "|"
  exit
fi
