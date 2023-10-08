package engine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"os"
)

// Docker 默认连接地址
const dockerUnix = "unix:///var/run/docker.sock"

type DockerCli struct {
	working      string         // 工作目录
	dockerClient *client.Client // Docker 镜像
	auth         string         // 授权信息
}

// NewDockerCli 创建一个构建引擎
func NewDockerCli(working string) (*DockerCli, error) {
	dockerClient, err := client.NewClientWithOpts(client.WithHost(dockerUnix),
		client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerCli{working: working, dockerClient: dockerClient}, nil
}

// NewAuthDockerCli 使用授权信息创建一个构建引擎
func NewAuthDockerCli(working, authUserName, authPassword string) (*DockerCli, error) {
	if authUserName == "" || authPassword == "" {
		return NewDockerCli(working)
	}
	dockerClient, err := client.NewClientWithOpts(client.WithHost(dockerUnix),
		client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	authJson, err := json.Marshal(types.AuthConfig{
		Username: authUserName,
		Password: authPassword,
	})
	if err != nil {
		return nil, err
	}
	return &DockerCli{working: working, dockerClient: dockerClient,
		auth: base64.URLEncoding.EncodeToString(authJson)}, nil
}

// DockerRemove 删除容器镜像
func (docker *DockerCli) DockerRemove(imageName string) error {
	imageList, err := docker.dockerClient.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return err
	}
	var imageId = func() *string {
		for _, image := range imageList {
			for _, it := range image.RepoTags {
				if it == imageName {
					return &image.ID
				}
			}
		}
		return nil
	}()
	if imageId == nil {
		return nil
	}
	_, err = docker.dockerClient.ImageRemove(context.Background(), *imageId, types.ImageRemoveOptions{
		Force: true,
	})
	if err != nil {
		return err
	}
	return nil
}

// DockerTag 删除容器重命名
func (docker *DockerCli) DockerTag(sourceImage, targetImage string) error {
	err := docker.dockerClient.ImageTag(context.Background(), sourceImage, targetImage)
	if err != nil {
		return err
	}
	return nil
}

// DockerPull 远程仓库拉取最新镜像
func (docker *DockerCli) DockerPull(image string) error {
	reader, err := docker.dockerClient.ImagePull(context.Background(), image, types.ImagePullOptions{
		RegistryAuth: docker.auth,
	})
	if err != nil {
		return err
	}
	defer func(reader io.ReadCloser) {
		_ = reader.Close()
	}(reader)
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}
	return nil
}

// DockerPush 推送镜像到远程仓库
func (docker *DockerCli) DockerPush(image string) error {
	reader, err := docker.dockerClient.ImagePush(context.Background(), image, types.ImagePushOptions{
		RegistryAuth: docker.auth,
	})
	if err != nil {
		return err
	}
	defer func(reader io.ReadCloser) {
		_ = reader.Close()
	}(reader)
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}
	return nil
}
