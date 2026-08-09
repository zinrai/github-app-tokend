#!/usr/bin/env python3
"""Stands in for an application that needs a GitHub App token.

It reads the file at the point of use and does nothing else, which is the whole
of what github-app-tokend asks of the programs it serves.
"""

import argparse
import json
import time
import urllib.error
import urllib.request


def log(message):
    print("%s %s" % (time.strftime("%Y-%m-%dT%H:%M:%S%z"), message), flush=True)


def call(token_file, url):
    # Read the token again every time. Holding onto the first one would work
    # for fifty minutes and then stop, which is the mistake this program exists
    # to not make.
    with open(token_file) as f:
        token = f.read().strip()

    request = urllib.request.Request(
        url,
        headers={
            "Authorization": "Bearer " + token,
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    with urllib.request.urlopen(request) as response:
        body = json.load(response)

    names = []
    for repository in body["repositories"]:
        names.append(repository["full_name"])

    # The token is deliberately not logged. Its last four characters are enough
    # to see that a renewal has been picked up.
    return token[-4:], names


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--token-file", required=True)
    parser.add_argument(
        "--url", default="https://api.github.com/installation/repositories"
    )
    parser.add_argument("--interval", type=int, default=15)
    args = parser.parse_args()

    while True:
        try:
            tail, names = call(args.token_file, args.url)
            log("call succeeded token_tail=%s repositories=%s" % (tail, names))
        except urllib.error.HTTPError as e:
            log("call failed: %s %s" % (e.code, e.read().decode().strip()))
        except OSError as e:
            log("call failed: %s" % e)
        time.sleep(args.interval)


if __name__ == "__main__":
    main()
