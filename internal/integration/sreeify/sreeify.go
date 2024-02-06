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
	chunkSize     = 1024 * 1024 // 1MB
	pingFrequency = 15 * time.Second
)

type Client struct {
	client pb.SreeificationServiceClient
	conn   pb.SreeificationService_SreeifyClient
	//tc     trafficcontroller.Controller[string, *pb.Payload]
	m map[string]chan []byte
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

	time.Sleep(5 * time.Second)
	err = c.createConnection()
	if err != nil {
		return nil, err
	}

	c.m = make(map[string]chan []byte)

	return c, nil
}

func (c *Client) Sreeify(input []byte) ([]byte, error) {
	rawId, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	id := rawId.String()

	// Chunk and send input
	go func() {
		chunks := chunkData(input)
		for i, chunk := range chunks {
			payload := &pb.Payload{
				Id:         id,
				Part:       int32(i),
				TotalParts: int32(len(chunks)),
				Data:       chunk,
			}
			err = c.conn.Send(&pb.Sreequest{
				Data: &pb.Sreequest_Payload{
					Payload: payload,
				},
			})
			if err != nil {
				slog.Error(fmt.Sprintf("Error sending chunk %d: %s", i, err))
			}
		}
		if err != nil {
			slog.Error(fmt.Sprintf("Error closing send: %s", err))
		}
	}()

	// Blocking call to receive response
	resp, err := c.receive(id)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) createConnection() error {
	ctx := context.Background()
	conn, err := c.client.Sreeify(ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("Error creating connection: %s", err))
		return err
	}
	c.conn = conn

	go runPing(conn)
	go c.runReceiver(conn)
	return nil
}

func runPing(conn pb.SreeificationService_SreeifyClient) {
	ticker := time.NewTicker(pingFrequency)

	for {
		select {
		case <-ticker.C:
			err := conn.Send(&pb.Sreequest{
				Data: &pb.Sreequest_Ping{
					Ping: &pb.Ping{
						Time: time.Now().UnixMicro(),
					},
				},
			})
			if err != nil {
				slog.Error(fmt.Sprintf("Error sending ping: %s", err))
			}
		}
	}
}

func (c *Client) runReceiver(conn pb.SreeificationService_SreeifyClient) {
	cc := make(chan *pb.Payload)
	go c.collect(cc)

	for {
		resp, err := conn.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error(fmt.Sprintf("error receiving message: %+v", err))
			return
		}

		data := resp.GetData()
		switch x := data.(type) {
		case *pb.Sreesponse_Ping:
			handlePing(x)
		case *pb.Sreesponse_Payload:
			cc <- x.Payload
		}
	}
}

func (c *Client) collect(cc <-chan *pb.Payload) {
	data := make(map[string][][]byte)

	f := func(b []byte) bool {
		return b != nil
	}

	for payload := range cc {
		id := payload.GetId()
		if _, ok := data[id]; !ok {
			data[id] = make([][]byte, payload.TotalParts)
		}
		data[id][payload.GetPart()] = payload.GetData()

		if all(data[id], f) {
			b := flatten(data[id])
			co := c.m[id]
			co <- b
		}
	}
}

func flatten(bs [][]byte) []byte {
	var b []byte
	for _, b2 := range bs {
		b = append(b, b2...)
	}
	return b
}

func all(i [][]byte, f func([]byte) bool) bool {
	for _, x := range i {
		if !f(x) {
			return false
		}
	}
	return true
}

func (c *Client) receive(id string) ([]byte, error) {
	pc := make(chan []byte)
	c.m[id] = pc
	data := <-pc
	return data, nil
}

func chunkData(b []byte) [][]byte {
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

func handlePing(ping *pb.Sreesponse_Ping) {
	end := time.Now()
	ts := ping.Ping.Time
	start := time.UnixMicro(ts)
	delta := end.Sub(start)
	slog.Info(fmt.Sprintf("ping took %s", delta))
}
