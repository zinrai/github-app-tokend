# github-app-tokend

Keeps a GitHub App installation access token in a file, rewriting it before it
expires.

Applications read the file when they need the token. They do not sign JWTs,
call the GitHub API, or hold the App private key.

A single static binary with no dependencies outside the standard library.

## Usage

```
github-app-tokend \
  -app-id 12345 \
  -private-key /etc/github-app-tokend/key.pem \
  -installation-id 87654321 \
  -out /run/github-app-tokend/token
```

`-app-id`, `-private-key`, `-installation-id` and `-out` are required, and
`github-app-tokend -h` lists the rest. The configuration is read from the
command line and nowhere else, so what a running process was asked to do is
visible in the process listing.

The installation ID is the last path segment of the App's installation settings
page on GitHub.

Applications read the file every time they need the token, not once at startup.
It is replaced while the token it holds is still valid. The file holds the
token followed by a newline.

On GitHub Enterprise Server, pass the REST API root that server documents.
`-api-base` is used as written and defaults to `https://api.github.com`.

```
-api-base https://ghe.example.jp/api/v3
```

## Guarantees

The first token is requested before the process enters its loop, so a
misconfiguration fails the start rather than leaving a daemon running with
nothing to show for it.

Renewal is requested with a third of the token's life left. A failed renewal is
logged and retried every thirty seconds, leaving the existing file untouched.
An outage at GitHub does not take away a credential that is still valid.

The file is written to a temporary file in the same directory and renamed into
place, mode 0600. A reader never sees a partial token.

## What this does not do

No narrowing. The token carries exactly what the installation grants, so the
App and the installation are where its reach is decided.

No authorization. Any process that can read the file gets the token, and where
`-out` points is what decides which processes those are.

No audit. Reads of the file are not recorded.

No expiry detection. If renewal keeps failing until the token expires, the file
holds a dead token and requests using it return 401. Watch the logs.

## License

This project is licensed under the [MIT License](LICENSE).
