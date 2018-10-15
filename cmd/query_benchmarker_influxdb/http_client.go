package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// HTTPClient is a reusable HTTP Client.
type DefaultHTTPClient struct {
	HTTPClientCommon
	client     *http.Client
}

// NewHTTPClient creates a new HTTPClient.
func NewDefaultHTTPClient(host string, debug int, dialTimeout time.Duration, readTimeout time.Duration, writeTimeout time.Duration) *DefaultHTTPClient {
	return &DefaultHTTPClient{
		client : &http.Client{
			Timeout: readTimeout, // TODO sets all timeouts
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: dialTimeout,
				}).Dial,
				MaxIdleConns: 1,
				MaxConnsPerHost: 1,
				IdleConnTimeout: 1 * time.Hour,
			},
		},
		HTTPClientCommon: HTTPClientCommon{
			Host:       []byte(host),
			HostString: host,
			debug:      debug,
		},
	}
}

// Do performs the action specified by the given Query. It uses fasthttp, and
// tries to minimize heap allocations.
func (w *DefaultHTTPClient) Do(q *Query, opts *HTTPClientDoOptions) (lag float64, err error) {
	// populate uri from the reusable byte slice:
	uri := make([]byte, 0, 100)
	uri = append(uri, w.Host...)
	uri = append(uri, q.Path...)

	// populate a request with data from the Query:
	req, err := http.NewRequest(string(q.Method), string(uri), bytes.NewBuffer(q.Body)) // TODO performance

	start := time.Now()
	resp, err := w.client.Do(req)
	lag = float64(time.Since(start).Nanoseconds()) / 1e6 // milliseconds

	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil || resp.StatusCode != http.StatusOK {
		values, _ := url.ParseQuery(string(uri))
		fmt.Printf("debug: url: %s, path %s, parsed url - %s\n", string(uri), q.Path, values)
	}

	// Check that the status code was 200 OK:
	if err == nil {
		sc := resp.StatusCode
		if sc != http.StatusOK {
			err = fmt.Errorf("Invalid write response (status %d): %s", sc, string(respBody))
			return
		}
	}

	if opts != nil {
		// Print debug messages, if applicable:
		switch opts.Debug {
		case 1:
			fmt.Fprintf(os.Stderr, "debug: %s in %7.2fms\n", q.HumanLabel, lag)
		case 2:
			fmt.Fprintf(os.Stderr, "debug: %s in %7.2fms -- %s\n", q.HumanLabel, lag, q.HumanDescription)
		case 3:
			fmt.Fprintf(os.Stderr, "debug: %s in %7.2fms -- %s\n", q.HumanLabel, lag, q.HumanDescription)
			fmt.Fprintf(os.Stderr, "debug:   request: %s\n", string(q.String()))
		case 4:
			fmt.Fprintf(os.Stderr, "debug: %s in %7.2fms -- %s\n", q.HumanLabel, lag, q.HumanDescription)
			fmt.Fprintf(os.Stderr, "debug:   request: %s\n", string(q.String()))
			fmt.Fprintf(os.Stderr, "debug:   response: %s\n", string(respBody))
		default:
		}

		// Pretty print JSON responses, if applicable:
		if opts.PrettyPrintResponses {
			// InfluxQL responses are in JSON and can be pretty-printed here.
			// Flux responses are just simple CSV.

			prefix := fmt.Sprintf("ID %d: ", q.ID)
			if json.Valid(respBody) {
				var pretty bytes.Buffer
				err = json.Indent(&pretty, respBody, prefix, "  ")
				if err != nil {
					return
				}

				_, err = fmt.Fprintf(os.Stderr, "%s%s\n", prefix, pretty)
				if err != nil {
					return
				}
			} else {
				_, err = fmt.Fprintf(os.Stderr, "%s%s\n", prefix, respBody)
				if err != nil {
					return
				}
			}
		}
	}

	return lag, err
}

func  (w *DefaultHTTPClient) HostString() string {
	return w.HTTPClientCommon.HostString
}

func (w *DefaultHTTPClient) Ping() {
	req, _ := http.NewRequest("GET", w.HTTPClientCommon.HostString + "/ping", nil)
	resp, _ := w.client.Do(req)
	defer resp.Body.Close()
}