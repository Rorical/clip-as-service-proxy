package proxy

import (
	"context"
	"errors"
	pb "github.com/Rorical/clip-as-service-proxy/encoder"
	"github.com/Rorical/clip-as-service-proxy/utils"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

var (
	wrongServiceType = errors.New("error wrong service type")
)

const (
	roundRobin = 0
)

const (
	textType  = 0
	imageType = 1
)

type EncodeRequest struct {
	TaskID string
	Data   interface{}
}

type EncodeResponse struct {
	TaskID string
	Point  []float32
	Err    error
}

type ClipServiceProxy struct {
	services        []pb.EncoderClient
	broadcast       *utils.Broadcast
	batch           *utils.BatchCollector
	storage         map[string]interface{}
	serviceType     int
	loadBalanceMode int
}

func encodeImage(cli pb.EncoderClient, ctx context.Context, images [][]byte) ([][]float32, error) {
	request := &pb.EncodeImageRequest{Images: images}
	response, err := cli.EncodeImage(ctx, request)
	if err != nil {
		return nil, err
	}
	batch := make([][]float32, len(response.GetEmbedding()))
	for i, b := range response.GetEmbedding() {
		batch[i] = b.Point
	}
	return batch, nil
}

func encodeText(cli pb.EncoderClient, ctx context.Context, texts []string) ([][]float32, error) {
	request := &pb.EncodeTextRequest{Texts: texts}
	response, err := cli.EncodeText(ctx, request)
	if err != nil {
		return nil, err
	}
	batch := make([][]float32, len(response.GetEmbedding()))
	for i, b := range response.GetEmbedding() {
		batch[i] = b.Point
	}
	return batch, nil
}

func (p *ClipServiceProxy) selectServer() pb.EncoderClient {
	serverIndex := 0
	switch p.loadBalanceMode {
	case roundRobin:
		serverIndex = p.storage["lastServerIndex"].(int)
		serverIndex += 1
		if serverIndex >= len(p.services) {
			serverIndex = 0
		}
		p.storage["lastServerIndex"] = serverIndex
	}
	return p.services[serverIndex]
}

func (p *ClipServiceProxy) prepareSelectServer() {
	switch p.loadBalanceMode {
	case roundRobin:
		p.storage["lastServerIndex"] = -1
		break
	}
}

func (p *ClipServiceProxy) batchProcessImages(items []interface{}) {
	ctx := context.TODO()

	images := make([][]byte, len(items))
	taskIds := make([]string, len(items))
	for i, g := range items {
		req := g.(EncodeRequest)
		images[i] = req.Data.([]byte)
		taskIds[i] = req.TaskID
	}
	s := p.selectServer()

	points, err := encodeImage(s, ctx, images)
	if err != nil {
		for _, d := range taskIds {
			p.broadcast.AddMessage(EncodeResponse{
				TaskID: d,
				Point:  nil,
				Err:    err,
			})
		}
		return
	}

	for i, d := range taskIds {
		p.broadcast.AddMessage(EncodeResponse{
			TaskID: d,
			Point:  points[i],
			Err:    nil,
		})
	}
	return
}

func (p *ClipServiceProxy) batchProcessTexts(items []interface{}) {
	ctx := context.TODO()

	texts := make([]string, len(items))
	taskIds := make([]string, len(items))
	for i, g := range items {
		req := g.(EncodeRequest)
		texts[i] = req.Data.(string)
		taskIds[i] = req.TaskID
	}

	s := p.selectServer()

	points, err := encodeText(s, ctx, texts)

	if err != nil {
		for _, d := range taskIds {
			p.broadcast.AddMessage(EncodeResponse{
				TaskID: d,
				Point:  nil,
				Err:    err,
			})
		}
		return
	}

	for i, d := range taskIds {
		p.broadcast.AddMessage(EncodeResponse{
			TaskID: d,
			Point:  points[i],
			Err:    nil,
		})
	}
	return
}

func (p *ClipServiceProxy) EncodeText(ctx context.Context, texts []string) ([][]float32, error) {
	if p.serviceType == imageType {
		return nil, wrongServiceType
	}
	data := make([]interface{}, len(texts))
	for i, d := range texts {
		data[i] = d
	}
	return p.encode(ctx, data)
}

func (p *ClipServiceProxy) EncodeImage(ctx context.Context, images [][]byte) ([][]float32, error) {
	if p.serviceType == textType {
		return nil, wrongServiceType
	}
	data := make([]interface{}, len(images))
	for i, d := range images {
		data[i] = d
	}
	return p.encode(ctx, data)
}

func (p *ClipServiceProxy) encode(ctx context.Context, data []interface{}) ([][]float32, error) {
	responseChannel := make(chan interface{}, len(data))
	p.broadcast.AddChannel(responseChannel)
	defer p.broadcast.RemoveChannel(responseChannel)

	resultPoints := make([][]float32, len(data))
	taskIds := make([]string, len(data))
	taskIdsMap := make(map[string]int)
	for i := range taskIds {
		taskIds[i] = uuid.New().String()
		taskIdsMap[taskIds[i]] = i
	}

	go func(batch *utils.BatchCollector, taskIds []string, data []interface{}) {
		for i, tid := range taskIds {
			req := EncodeRequest{
				TaskID: tid,
				Data:   data[i],
			}
			batch.Add(req)
		}
	}(p.batch, taskIds, data)

	counter := len(taskIds)
	for {
		select {
		case rawRes := <-responseChannel:
			res := rawRes.(EncodeResponse)
			if index, exists := taskIdsMap[res.TaskID]; exists {
				if res.Err != nil {
					return nil, res.Err
				}

				resultPoints[index] = res.Point

				counter -= 1
				if counter == 0 {
					return resultPoints, nil
				}
			}
		case <-ctx.Done():
			return nil, nil
		}
	}
}

func NewClipServiceProxy(config *Config) (*ClipServiceProxy, error) {
	var conn *grpc.ClientConn
	var err error
	encoderClients := make([]pb.EncoderClient, len(config.ClipServiceURIs))
	for i, uri := range config.ClipServiceURIs {
		conn, err = grpc.Dial(uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		encoderClients[i] = pb.NewEncoderClient(conn)
	}

	broadcast := utils.NewBroadcast()

	storage := make(map[string]interface{})
	p := &ClipServiceProxy{
		services:        encoderClients,
		broadcast:       broadcast,
		serviceType:     config.ServiceType,
		loadBalanceMode: config.LoadBalanceMode,
		storage:         storage,
	}
	p.prepareSelectServer()
	var batchFn func(items []interface{})
	if p.serviceType == textType {
		batchFn = p.batchProcessTexts
	} else if p.serviceType == imageType {
		batchFn = p.batchProcessImages
	}
	p.batch = utils.NewBatchCollector(config.TargetBatchSize, batchFn, time.Duration(config.TargetBatchTimeout)*time.Second)

	return p, nil
}
