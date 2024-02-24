package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"golang.org/x/net/http2"
)

type Client struct {
	c *http.Client
}

func NewClient(certFile, keyFile string) (*Client, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	transport := &http2.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	return &Client{
		c: &http.Client{
			Transport: transport,
		},
	}, nil
}

type Notification struct {
	Token     string
	PushMagic string
}

func (c *Client) Push(ctx context.Context, notifications []*Notification) map[string]error {
	type result struct {
		token string
		err   error
	}

	// TODO: reconsider if a worker pool should be used. This naively
	// assumes that the number of notifications sent will never overwhelm
	// the server.
	respChan := make(chan *result, len(notifications))
	defer close(respChan)
	out := make(map[string]error, len(notifications))
	for _, n := range notifications {
		n := n
		go func() {
			respChan <- &result{
				token: n.Token,
				err:   c.push(ctx, n),
			}
		}()
	}

	for range notifications {
		res := <-respChan
		out[res.token] = res.err
	}

	return out
}

func (c *Client) push(ctx context.Context, notification *Notification) error {
	payload := []byte(`{"mdm":"` + notification.PushMagic + `"}`)
	url := fmt.Sprintf("https://api.push.apple.com/3/device/%s", notification.Token)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response, err := c.c.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode >= http.StatusOK {
		return fmt.Errorf("unexpected status code %d", response.StatusCode)
	}
	return nil
}
