package memos

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/sirupsen/logrus"
	memosV1pb "github.com/usememos/memos/proto/gen/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/pirogoeth/apps/voice-memos/types"
)

type Client struct {
	ctx              context.Context
	WorkspaceService memosV1pb.WorkspaceServiceClient
	AuthService      memosV1pb.AuthServiceClient
	UserService      memosV1pb.UserServiceClient
	MemoService      memosV1pb.MemoServiceClient
	ResourceService  memosV1pb.ResourceServiceClient
}

func New(ctx context.Context, cfg types.MemosServerCfg) (*Client, error) {
	var options []grpc.DialOption
	if cfg.GrpcInsecure {
		logrus.Infof("Using insecure credentials to connect to Memos!")
		options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		logrus.Infof("Using system x509 pool to connect to Memos!")
		certPool, err := x509.SystemCertPool()
		if err != nil {
			logrus.Fatalf("failed to get system cert pool: %v", err)
		}

		creds := credentials.NewClientTLSFromCert(certPool, "")
		options = append(options, grpc.WithTransportCredentials(creds))
	}

	conn, err := grpc.NewClient(cfg.GrpcEndpoint, options...)
	if err != nil {
		return nil, fmt.Errorf("creating grpc client connection: %w", err)
	}

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"Authorization", fmt.Sprintf("Bearer %s", cfg.ApiToken),
	))

	client := &Client{
		ctx:              ctx,
		WorkspaceService: memosV1pb.NewWorkspaceServiceClient(conn),
		AuthService:      memosV1pb.NewAuthServiceClient(conn),
		UserService:      memosV1pb.NewUserServiceClient(conn),
		MemoService:      memosV1pb.NewMemoServiceClient(conn),
		ResourceService:  memosV1pb.NewResourceServiceClient(conn),
	}

	// Validate API token
	user, err := client.AuthService.GetAuthStatus(ctx, &memosV1pb.GetAuthStatusRequest{})
	if err != nil {
		return nil, fmt.Errorf("checking memos API access token: %w", err)
	}

	logrus.Infof("memos logged in as %s", user.Username)

	return client, nil
}

func (c *Client) Context() context.Context {
	return c.ctx
}
