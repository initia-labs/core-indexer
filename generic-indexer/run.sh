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

  go build -o generic-indexer.bin .

  source .env

  ./generic-indexer.bin sweep --bootstrap-server pkc-ldvr1.asia-southeast1.gcp.confluent.cloud:9092 \
    --block-topic ${chain}-local-generic-indexer-block-messages \
    --kafka-api-key $KAFKA_API_KEY \
    --kafka-api-secret $KAFKA_API_SECRET \
    --aws-access-key $AWS_ACCESS_KEY \
    --aws-secret-key $AWS_SECRET_KEY \
    --claim-check-bucket ${chain}-local-generic-indexer-large-block-messages \
    --claim-check-threshold-mb 1 \
    --db $DB_CONNECTION_STRING \
    --chain $chain \
    --poll-interval 500 \
    --rebalance-interval $REBALANCE_INTERVAL \
    --workers 4
}

help__flush="flush <..args> : run flush"
task__flush() {
  local chain=$1
  local id=$2

  if [ -z "$chain" ]; then
    echo "usage: $0 flush <chain> <id>"
    exit
  fi

  if [ -z "$id" ]; then
    echo "usage: $0 flush <chain> <id>"
    exit
  fi

  go build -o generic-indexer.bin .

  source .env

  ./generic-indexer.bin flush --bootstrap-server pkc-ldvr1.asia-southeast1.gcp.confluent.cloud:9092 \
    --block-topic ${chain}-local-generic-indexer-block-messages \
    --tx-topic ${chain}-local-lcd-tx-response-messages \
    --kafka-api-key $KAFKA_API_KEY \
    --kafka-api-secret $KAFKA_API_SECRET \
    --block-consumer-group ${chain}-local-generic-indexer-flusher \
    --aws-access-key $AWS_ACCESS_KEY \
    --aws-secret-key $AWS_SECRET_KEY \
    --block-claim-check-bucket ${chain}-large-generic-block-messages \
    --claim-check-threshold-mb 1 \
    --db $DB_CONNECTION_STRING \
    --chain $chain \
    --disable-lcd-tx-response \
    --environment local \
    --rebalance-interval $REBALANCE_INTERVAL \
    --id $id
}

help__validatorcron="validatorcron <..args> : run validatorcron"
task__validatorcron() {
  local chain=$1
  local id=$2

  if [ -z "$chain" ]; then
    echo "usage: $0 validatorcron <chain> <id>"
    exit
  fi

  if [ -z "$id" ]; then
    echo "usage: $0 validatorcron <chain> <id>"
    exit
  fi

  go build -o generic-indexer.bin .

  source .env

  ./generic-indexer.bin validatorcron  --db $DB_CONNECTION_STRING \
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
