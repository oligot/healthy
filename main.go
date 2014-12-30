package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/user"
	"strconv"
	"text/template"
	"time"

	"github.com/docopt/docopt-go"
)

const Version = "0.0.1"

const Usage = `Send a mail when services are unavailable

  Usage:
    healthy <interval> [--es-host host] [--es-search json] [--from mail] [--to mail] [--smtp-host host]
    healthy -h | --help
    healthy --version

  Options:
    --es-host host   	ElasticSearch host (default to localhost)
    --es-search json   	JSON file used as request body (default to 'response:5*')
    --from mail   	mail sender (default to healthy@$HOST)
    --to mail   	mail recipient (default to $USER@$HOST)
    --smtp-host host   	SMTP host (default to localhost)
    -h, --help          output help information
    -v, --version       output version

`

// Search response
type Search struct {
	Hits struct {
		Total int64
		Hits  []struct {
			Source struct {
				Message string
			} `json:"_source"`
		}
	}
}

// Search template
type SearchTemplate struct {
	Interval string
}

type Options struct {
	esHost   string
	body     bytes.Buffer
	from     string
	to       string
	smtpHost string
}

func main() {
	args, err := docopt.Parse(Usage, nil, true, Version, false)
	if err != nil {
		log.Fatalf("failed to parse arguments: %s", err)
	}
	interval := args["<interval>"].(string)
	duration, err := time.ParseDuration(interval)
	if err != nil {
		log.Fatalf("failed to parse duration: %s", err)
	}
	esHost := option(args, "--es-host", "localhost")
	var body bytes.Buffer
	if value, ok := args["--es-search"].(string); ok {
		data, err := ioutil.ReadFile(value)
		if err != nil {
			log.Fatalf("failed to read file: %s", err)
		}
		tmpl, err := template.New("search").Parse(string(data))
		if err != nil {
			log.Fatalf("failed to parse template: %s", err)
		}
		err = tmpl.Execute(&body, SearchTemplate{interval})
		if err != nil {
			log.Fatalf("failed to execute template: %s", err)
		}
	}
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %s", err)
	}
	u, err := user.Current()
	if err != nil {
		log.Fatalf("failed to get user: %s", err)
	}
	from := option(args, "--from", "healthy@"+hostname)
	to := option(args, "--to", u.Username+"@"+hostname)
	smtpHost := option(args, "--smtp-host", "localhost")
	options := Options{esHost, body, from, to, smtpHost}

	for {
		log.Printf("waiting %s", duration)
		time.Sleep(duration)
		search(options)
	}
}

// Search unavailable services
func search(options Options) {
	now := time.Now().Format("2006.01.02")
	url := "http://" + options.esHost + ":9200/logstash-" + now + "/_search"
	var res *http.Response
	var err error
	if options.body.Len() > 0 {
		log.Printf("searching body %s", url)
		res, err = http.Post(url, "application/json", &options.body)
	} else {
		url += "?q=response:5*"
		log.Printf("searching URI %s", url)
		res, err = http.Get(url)
	}
	if err != nil {
		log.Fatalf("request failed: %s", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("request error: %s", http.StatusText(res.StatusCode))
	}

	var search Search
	if err := json.NewDecoder(res.Body).Decode(&search); err != nil {
		log.Fatalf("failed to decode response: %s", err)
	}
	log.Printf("found %v hits", search.Hits.Total)
	if search.Hits.Total > 0 {
		var messages bytes.Buffer
		for _, hit := range search.Hits.Hits {
			messages.WriteString(hit.Source.Message + "\n")
		}
		log.Printf("messages: %s", messages.String())
		send(options, search.Hits.Total, messages.String())
	}
}

// Send mail
func send(options Options, total int64, messages string) {
	log.Printf("sending mail from %s to %s using %s", options.from, options.to, options.smtpHost)
	to := []string{options.to}
	msg := []byte("Subject: " + strconv.FormatInt(total, 10) + ` service(s) unavailable

` + messages)
	err := smtp.SendMail(options.smtpHost+":25", nil, options.from, to, msg)
	if err != nil {
		log.Fatalf("failed to send mail: %s", err)
	}
}

// Get the option value, with a fallback to a default value if not found
func option(args map[string]interface{}, name string, defaultValue string) string {
	if value, ok := args[name].(string); ok {
		return value
	} else {
		return defaultValue
	}
}
