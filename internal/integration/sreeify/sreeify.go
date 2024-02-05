package sreeify

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/devhou-se/sreetcode/internal/config"
	pb "github.com/devhou-se/sreetcode/internal/gen"
)

const (
	chunkSize = 1024 * 1024 // 1MB
)

type Client struct {
	client pb.SreeificationServiceClient
	conn   pb.SreeificationService_SreeifyClient
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

	bo := backoff.Config{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Multiplier: 1.6,
		Jitter:     0.2,
	}

	opts := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: bo}),
	}

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

	conn, err := c.client.Sreeify(ctx)
	if err != nil {
		return nil, err
	}

	rawId, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	id := rawId.String()

	// Chunk and send input
	go func() {
		chunks := chunk(input)
		for i, c := range chunks {
			err = conn.Send(&pb.Sreequest{
				Id:         id,
				Part:       int32(i),
				TotalParts: int32(len(chunks)),
				Data:       c,
			})
			if err != nil {
				slog.Error(fmt.Sprintf("Error sending chunk %d: %s", i, err))
			}
		}
		err = conn.CloseSend()
		if err != nil {
			slog.Error(fmt.Sprintf("Error closing send: %s", err))
		}
	}()

	// Blocking call to receive response
	resp, err := receive(conn)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func chunk(b []byte) [][]byte {
	var bs [][]byte
	for i := 0; i < len(b); i += chunkSize {
		end := i + chunkSize
		if end > len(b) {
			end = len(b)
		}
		bs = append(bs, b[i:end])
	}
	return bs
}

func receive(conn pb.SreeificationService_SreeifyClient) ([]byte, error) {
	var bs [][]byte
	received := 0
	for {
		resp, err := conn.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if int32(len(bs)) != resp.TotalParts {
			bs2 := make([][]byte, resp.TotalParts)
			copy(bs2, bs)
			bs = bs2
		}

		bs[resp.Part] = resp.Data
		received++
	}

	var b []byte
	for _, part := range bs {
		b = append(b, part...)
	}

	return b, nil
}
