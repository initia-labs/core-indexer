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


create(f"topics/{chain}-local-block-results", {})
create(f"topics/{chain}-local-tx-responses", {})
create(f"subscriptions/{chain}-local-informative-indexer-indexer", {
    "topic": f"projects/{os.environ['PUBSUB_PROJECT_ID']}/topics/{chain}-local-block-results",
    "enableMessageOrdering": True,
})
