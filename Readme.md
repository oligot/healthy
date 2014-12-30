
# healthy

 Send a mail when services are unavailable

## Installation

```
$ go get github.com/oligot/healthy
```

## Usage

```

  Usage:
    healthy <interval> [--es-host host] [--es-search json] [--from mail] [--to mail] [--smtp-host host]
    healthy -h | --help
    healthy --version

  Options:
    --es-host host      ElasticSearch host (default to localhost)
    --es-search json    JSON file used as request body (default to 'response:5*')
    --from mail         mail sender (default to healthy@$HOST)
    --to mail           mail recipient (default to $USER@$HOST)
    --smtp-host host    SMTP host (default to localhost)
    -h, --help          output help information
    -v, --version       output version

```

## Example

  Check each minute to see if services are unavailable

```
$ healthy 1m
```

# License

MIT
