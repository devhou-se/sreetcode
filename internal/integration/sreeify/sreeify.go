package sreeify

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/devhou-se/sreetcode/internal/config"
	pb "github.com/devhou-se/sreetcode/internal/gen"
)

type Client struct {
	client pb.SreeificationServiceClient
}

func loadTLS() grpc.DialOption {
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		panic(err)
	}
	creds := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})
	return grpc.WithTransportCredentials(creds)
}

func NewClient(cfg config.Config) (*Client, error) {
	c := &Client{}

	var opts []grpc.DialOption

	if cfg.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, loadTLS())
	}

	conn, err := grpc.Dial(cfg.SreeifierServer, opts...)
	if err != nil {
		return nil, err
	}

	c.client = pb.NewSreeificationServiceClient(conn)

	return c, nil
}

func (c *Client) Sreeify(input []byte) ([]byte, error) {
	ctx := context.Background()

	inpString := string(input)

	req := &pb.SreeifyRequest{
		LinkReplacements: []*pb.SreeifyRequest_LinkReplacement{
			{OriginalBaseUri: "en.wikipedia.org", ReplacementBaseUri: "en.sreekipedia.org"},
		},
		Data: &pb.SreeifyRequest_Payload{
			Payload: inpString,
		},
	}

	start := time.Now()
	resp, err := c.client.Sreeify(ctx, req)
	slog.Info(fmt.Sprintf("Sreeify took %s", time.Since(start)))
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	return []byte(resp.GetPayload()), nil
}
