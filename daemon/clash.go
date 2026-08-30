package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Клиент Clash API sing-box: источник трафика и лога соединений.

type clashConn struct {
	ID          string   `json:"id"`
	Upload      int64    `json:"upload"`
	Download    int64    `json:"download"`
	Start       string   `json:"start"`
	Chains      []string `json:"chains"`
	Rule        string   `json:"rule"`
	RulePayload string   `json:"rulePayload"`
	Metadata    struct {
		Network         string `json:"network"`
		Host            string `json:"host"`
		DestinationIP   string `json:"destinationIP"`
		DestinationPort string `json:"destinationPort"`
	} `json:"metadata"`
}

type clashConns struct {
	DownloadTotal int64       `json:"downloadTotal"`
	UploadTotal   int64       `json:"uploadTotal"`
	Connections   []clashConn `json:"connections"`
}

type clashClient struct {
	http *http.Client
}

func newClashClient() *clashClient {
	return &clashClient{http: &http.Client{Timeout: 3 * time.Second}}
}

func (c *clashClient) get(path string, v any) error {
	resp, err := c.http.Get("http://" + ClashAddr + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("clash %s: http %d", path, resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if v == nil {
		return nil
	}
	return json.Unmarshal(b, v)
}

// Alive проверяет, поднялся ли sing-box.
func (c *clashClient) Alive() bool { return c.get("/version", nil) == nil }

func (c *clashClient) Conns() (*clashConns, error) {
	var out clashConns
	if err := c.get("/connections", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
