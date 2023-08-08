package proxyservice

import (
	"context"
	pb "github.com/Rorical/clip-as-service-proxy/encoder"
	proxy2 "github.com/Rorical/clip-as-service-proxy/proxy"
	"google.golang.org/grpc"
	"net"
)

type ProxyService struct {
	proxy *proxy2.ClipServiceProxy
	pb.UnimplementedEncoderServer
}

func (p *ProxyService) EncodeText(ctx context.Context, request *pb.EncodeTextRequest) (*pb.EncoderResponse, error) {
	points, err := p.proxy.EncodeText(ctx, request.Texts)
	if err != nil {
		return nil, err
	}
	embeddings := make([]*pb.Embedding, len(request.Texts))
	for i, p := range points {
		embeddings[i] = &pb.Embedding{Point: p}
	}
	return &pb.EncoderResponse{
		Embedding: embeddings,
	}, nil
}

func (p *ProxyService) EncodeImage(ctx context.Context, request *pb.EncodeImageRequest) (*pb.EncoderResponse, error) {
	points, err := p.proxy.EncodeImage(ctx, request.Images)
	if err != nil {
		return nil, err
	}
	embeddings := make([]*pb.Embedding, len(request.Images))
	for i, p := range points {
		embeddings[i] = &pb.Embedding{Point: p}
	}
	return &pb.EncoderResponse{
		Embedding: embeddings,
	}, nil
}

func Serve(addr string, configPath string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	config, err := proxy2.ReadConfig(configPath)
	if err != nil {
		return err
	}
	proxy, err := proxy2.NewClipServiceProxy(config)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterEncoderServer(s, &ProxyService{
		proxy,
		pb.UnimplementedEncoderServer{},
	})
	if err = s.Serve(listener); err != nil {
		return err
	}
	return nil
}
