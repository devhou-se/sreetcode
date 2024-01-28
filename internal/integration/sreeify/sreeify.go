package sreeify

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/devhou-se/sreetcode/internal/config"
	pb "github.com/devhou-se/sreetcode/internal/gen"
)

type Client struct {
	client pb.SreeificationServiceClient
}

func NewClient(cfg config.Config) (*Client, error) {
	c := &Client{}

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

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

	resp, err := c.client.Sreeify(ctx, req)
	if err != nil {
		return nil, err
	}

	return []byte(resp.GetPayload()), nil
}
