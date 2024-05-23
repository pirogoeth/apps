package client

import (
	"context"
	"encoding/json"

	"github.com/imroc/req/v3"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/sirupsen/logrus"
)

type Options struct {
	BaseURL     string `json:"base_url"`
	DevMode     bool   `json:"dev_mode"`
	WorkerToken string `json:"worker_token"`
}

type Client struct {
	httpClient *req.Client
}

type commonResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type NetworksResponse struct {
	commonResponse
	Networks []database.Network `json:"networks,omitempty"`
}

type HostsResponse struct {
	commonResponse
	Hosts []database.Host `json:"hosts,omitempty"`
}

func NewClient(opts *Options) *Client {
	cli := &Client{
		httpClient: req.NewClient(),
	}

	cli.httpClient.BaseURL = opts.BaseURL

	if opts.DevMode {
		cli.httpClient = cli.httpClient.DevMode()
	}

	if opts.WorkerToken != "" {
		cli.httpClient.SetCommonBearerAuthToken(opts.WorkerToken)
	}

	return cli
}

func (c *Client) ListNetworks(ctx context.Context) (*NetworksResponse, error) {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get("/v1/networks")
	if err != nil {
		return nil, err
	}

	ret := &NetworksResponse{}
	if err := json.Unmarshal(resp.Bytes(), ret); err != nil {
		logrus.Errorf("could not unmarshal response: %s", err)
		return nil, err
	}

	return ret, nil
}

func (c *Client) CreateHost(ctx context.Context, hostParams *database.CreateHostParams) (*HostsResponse, error) {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBodyJsonMarshal(hostParams).
		Post("/v1/hosts")
	if err != nil {
		return nil, err
	}

	ret := &HostsResponse{}
	if err := json.Unmarshal(resp.Bytes(), ret); err != nil {
		logrus.Errorf("could not unmarshal response: %s", err)
		return nil, err
	}

	return ret, nil
}
