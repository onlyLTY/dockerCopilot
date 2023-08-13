package utiles

import (
	"context"
	"encoding/json"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	dockerMsgType "github.com/docker/docker/pkg/jsonmessage"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	myTypes "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"log"
)

func GetNewImage(ctx *svc.ServiceContext, imageNameAndTag string) (myTypes.MsgResp, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %s", err)
	}

	// Ensure the client is compatible with the server version
	cli.NegotiateAPIVersion(context.TODO())
	if err != nil {
		return myTypes.MsgResp{}, err
	}
	reader, err := cli.ImagePull(context.TODO(), imageNameAndTag, dockerTypes.ImagePullOptions{})
	if err != nil {
		log.Fatalf("Failed to pull image: %s", err)
	}
	defer reader.Close()

	decodePullResponse(reader)
	return myTypes.MsgResp{}, nil
}

func decodePullResponse(reader io.Reader) {
	decoder := json.NewDecoder(reader)
	for {
		var msg dockerMsgType.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Failed to decode pull image response: %s", err)
		}

		// Print the progress or error information from the response
		if msg.Error != nil {
			logx.Error("Error: %s", msg.Error)
		} else {
			logx.Info("%s: %s\n", msg.Status, msg.Progress)
		}
	}
}
