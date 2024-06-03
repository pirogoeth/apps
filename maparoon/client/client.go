package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/imroc/req/v3"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/sirupsen/logrus"
)

var (
	ErrAlreadyExists = errors.New("resource already exists")
	ErrNotFound      = errors.New("resource not found")
	ErrInternal      = errors.New("upstream internal server error")
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

type HostPortsResponse struct {
	commonResponse
	HostPorts []database.HostPort `json:"host_ports,omitempty"`
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

	switch resp.StatusCode {
	case http.StatusConflict:
		return nil, fmt.Errorf("%w: %s", ErrAlreadyExists, ret.Error)
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: %s", ErrInternal, ret.Error)
	}

	return ret, nil
}

func (c *Client) CreateHostPort(ctx context.Context, hostPortParams *database.CreateHostPortParams) (*HostPortsResponse, error) {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBodyJsonMarshal(hostPortParams).
		SetPathParam("host_address", hostPortParams.Address).
		Post("/v1/host/{host_address}/ports")
	if err != nil {
		return nil, err
	}

	ret := &HostPortsResponse{}
	if err := json.Unmarshal(resp.Bytes(), ret); err != nil {
		logrus.Errorf("could not unmarshal response: %s", err)
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusConflict:
		return nil, fmt.Errorf("%w: %s", ErrAlreadyExists, ret.Error)
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: %s", ErrInternal, ret.Error)
	}

	return ret, nil
}

func (c *Client) CreateHostScans(ctx context.Context, hostScansReq types.CreateHostScansRequest) (*commonResponse, error) {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBodyJsonMarshal(hostScansReq).
		Post("/v1/hostscans")
	if err != nil {
		return nil, err
	}

	if resp.IsErrorState() {
		return nil, fmt.Errorf("error: %s", resp.String())
	}

	ret := &commonResponse{}
	if err := json.Unmarshal(resp.Bytes(), ret); err != nil {
		logrus.Errorf("could not unmarshal response: %s", err)
		return nil, err
	}

	return ret, nil
}
