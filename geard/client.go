package geard

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	defaultBufferSize = 10
)

type Client struct {
	Host       string
	Port       string
	url        string
	httpClient  *http.Client
	persistence io.Writer
	cURLch      chan string
	// CheckRetry can be used to control the policy for failed requests
	// and modify the cluster if needed.
	// The client calls it before sending requests again, and
	// stops retrying if CheckRetry returns some error. The cases that
	// this function needs to handle include no response and unexpected
	// http status code of response.
	// If CheckRetry is nil, client will call the default one
	// `DefaultCheckRetry`.
	// Argument cluster is the geard.Cluster object that these requests have been made on.
	// Argument numReqs is the number of http.Requests that have been made so far.
	// Argument lastResp is the http.Responses from the last request.
	// Argument err is the reason of the failure.
	CheckRetry func(numReqs int,
		lastResp http.Response, err error) error
}

// NewClient create a basic client that is configured to be used
// with the given machine list.
func NewClient(host string, port string) *Client {
	client := &Client{Host: host, Port: port}

	client.initHTTPClient()
	return client
}

// Override the Client's HTTP Transport object
func (c *Client) SetTransport(tr *http.Transport) {
	c.httpClient.Transport = tr
}

// initHTTPClient initializes a HTTP client for geard client
func (c *Client) initHTTPClient() {
	tr := &http.Transport{
		Dial: c.dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	c.httpClient = &http.Client{Transport: tr}
}


// dial attempts to open a TCP connection to the provided address, explicitly
// enabling keep-alives with a one-second interval.
func (c *Client) dial(network, addr string) (net.Conn, error) {       
        conn, err := net.Dial(network, addr)
	//conn, err := net.DialTimeout(network, addr, c.config.DialTimeout)
	if err != nil {
		return nil, err
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, errors.New("Failed type-assertion of net.Conn as *net.TCPConn")
	}

	// Keep TCP alive to check whether or not the remote machine is down
	if err = tcpConn.SetKeepAlive(true); err != nil {
		return nil, err
	}

	if err = tcpConn.SetKeepAlivePeriod(time.Second); err != nil {
		return nil, err
	}

	return tcpConn, nil
}

func (c *Client) OpenCURL() {
	c.cURLch = make(chan string, defaultBufferSize)
}

func (c *Client) CloseCURL() {
	c.cURLch = nil
}

func (c *Client) sendCURL(command string) {
	go func() {
		select {
		case c.cURLch <- command:
		default:
		}
	}()
}

func (c *Client) RecvCURL() string {
	return <-c.cURLch
}


