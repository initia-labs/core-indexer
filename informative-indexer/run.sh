#!/bin/bash

set -e

SCRIPT_DIR=$(cd $(dirname $0) ; pwd -P)

TASK=$1
ARGS=${@:2}

help__sweep="sweep <..args> : run sweep"
task__sweep() {
  local chain=$1

  if [ -z "$chain" ]; then
    echo "usage: $0 sweep <chain>"
    exit
  fi

  go build -o informative-indexer.bin .

  source .env

  ./informative-indexer.bin sweep --bootstrap-server $KAFKA_BOOTSTRAP_SERVER \
    --block-results-topic ${chain}-local-informative-indexer-block-results-messages \
    --kafka-api-key $KAFKA_API_KEY \
    --kafka-api-secret $KAFKA_API_SECRET \
    --claim-check-bucket ${chain}-local-informative-indexer-large-block-results \
    --claim-check-threshold-mb 1 \
    --db $DB_CONNECTION_STRING \
    --chain $chain \
    --rebalance-interval $REBALANCE_INTERVAL \
    --workers 4 \
    --migrations-dir ../db/migrations
}

help__flush="flush <..args> : run flush"
task__flush() {
  local chain=$1

  if [ -z "$chain" ]; then
    echo "usage: $0 flush <chain> <id>"
    exit
  fi

  go build -o informative-indexer.bin .

  source .env

  ./informative-indexer.bin flush --bootstrap-server $KAFKA_BOOTSTRAP_SERVER \
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

help__prune="prune <..args> : run prune"
task__prune() {
  local chain=$1

  if [ -z "$chain" ]; then
    echo "usage: $0 prune <chain>"
    exit
  fi

  go build -o informative-indexer.bin .

  source .env

  ./informative-indexer.bin prune --db $DB_CONNECTION_STRING \
    --backup-bucket-name ${chain}-local-core-informative-data-backup \
    --backup-file-prefix events \
    --pruning-keep-block 10 \
    --pruning-interval 1 \
    --chain $chain
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
