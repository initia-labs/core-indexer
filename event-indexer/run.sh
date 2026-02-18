#!/bin/bash

set -e

SCRIPT_DIR=$(cd $(dirname $0) ; pwd -P)

TASK=$1
ARGS=${@:2}


help__run="run <..args> : run run"
task__run() {
  local chain=$1

  if [ -z "$chain" ]; then
    echo "usage: $0 run <chain> <id>"
    exit
  fi

  go build -o event-indexer .

  source .env

  ./event-indexer run --bootstrap-server $KAFKA_BOOTSTRAP_SERVER \
    --block-results-topic ${chain}-local-generic-indexer-block-results-messages \
    --kafka-api-key $KAFKA_API_KEY \
    --kafka-api-secret $KAFKA_API_SECRET \
    --block-results-consumer-group ${chain}-local-event-indexer \
    --block-results-claim-check-bucket ${chain}-local-event-indexer-large-block-results \
    --claim-check-threshold-mb 1 \
    --db $DB_CONNECTION_STRING \
    --chain $chain \
    --id 1
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
