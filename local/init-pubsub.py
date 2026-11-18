"""Create resources in the local Pub/Sub emulator before workers start."""
import json
import os
import urllib.error
import urllib.request

base = f"http://{os.environ['PUBSUB_EMULATOR_HOST']}/v1/projects/{os.environ['PUBSUB_PROJECT_ID']}"
chain = os.environ['CHAIN']


def create(resource, body):
    request = urllib.request.Request(
        f"{base}/{resource}", data=json.dumps(body).encode(), method="PUT",
        headers={"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(request) as response:
            response.read()
    except urllib.error.HTTPError as error:
        if error.code != 409:
            raise


for suffix in ("sequential-indexer-block-results", "concurrent-indexer-block-results", "tx-responses"):
    create(f"topics/{chain}-{suffix}", {})

for component in ("generic-indexer-block-results", "event-indexer-block-results", "tx-responses"):
    create(f"topics/dlq-{chain}-{component}", {"messageRetentionDuration": "1296000s"})

for component, topic, ordered in (
    ("generic-indexer", "concurrent-indexer-block-results", False),
    ("event-indexer", "concurrent-indexer-block-results", False),
    ("informative-indexer", "sequential-indexer-block-results", True),
    ("tx-response-uploader", "tx-responses", False),
):
    create(f"subscriptions/{chain}-{component}", {
        "topic": f"projects/{os.environ['PUBSUB_PROJECT_ID']}/topics/{chain}-{topic}",
        "enableMessageOrdering": ordered,
        "ackDeadlineSeconds": 60,
        "messageRetentionDuration": "604800s",
        "expirationPolicy": {"ttl": ""},
    })
