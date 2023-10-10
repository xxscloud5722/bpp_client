package console

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/nuwa/bpp.v3/engine"
	"github.com/nuwa/bpp.v3/environment"
	"github.com/nuwa/bpp.v3/message"
	"path"
	"strings"
)

/*
 * 环境变量 :
 *  - CI 开头是GitLab 的环境变量
 *  - GL 开头全局环境变量
 *  - GS 开头是证书文件, 存放在Key服务器只有用的时候才会获取
 *  - P  开头是CI 文件写的环境变量
 *  - GO 开头是程序运行产生的环境变量
 *  - 驼峰的环境变量 是程序运行的时候入参 也有可能是早期程序运行产生的环境变量
 */

// dockerAuth 容器授权参数.
func dockerAuth() (*string, *string, error) {
	// 读取镜像授权类型
	authType, ok := environment.Get("P_DOCKER_AUTH_TYPE")
	authType = strings.ToUpper(authType)
	username := ""
	password := ""
	if ok {
		var authDockerValue string
		authDockerValue, ok = environment.Get("GL_DOCKER_AUTH_" + authType)
		if !ok {
			return nil, nil, errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "GL_DOCKER_AUTH_"+authType))
		}
		username, password = func() (string, string) {
			var values = strings.Split(authDockerValue, ",")
			if len(values) > 1 {
				return values[0], values[1]
			}
			return "", ""
		}()
		if username == "" || password == "" {
			return nil, nil, errors.New(fmt.Sprintf("Environment variable ${%s - %s} not exist", "GL_DOCKER_AUTH", authType))
		}
	}
	return &username, &password, nil
}

// Package 构建打包.
func Package() error {
	// 读取工作目录
	working, ok := environment.Get("CI_PROJECT_DIR")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_PROJECT_DIR"))
	}
	// 读取项目ID
	projectId, ok := environment.Get("CI_PROJECT_ID")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_PROJECT_ID"))
	}
	// 读取构建类型
	packageType, ok := environment.Get("P_PACKAGE_TYPE")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_PACKAGE_TYPE"))
	}
	packageType = strings.ToUpper(packageType)
	var scriptEngine = engine.NewScriptEngine(working)
	err := scriptEngine.CreateBuild(projectId, packageType)
	if err != nil {
		return err
	}
	err = scriptEngine.CreateConfigFile(packageType)
	if err != nil {
		return err
	}
	err = scriptEngine.CreateDockerFile(packageType)
	if err != nil {
		return err
	}
	return nil
}

// DockerPush 容器镜像获取.
func DockerPush() error {
	// 读取工作目录
	working, ok := environment.Get("CI_PROJECT_DIR")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_PROJECT_DIR"))
	}
	// 读取镜像名称
	imageName, ok := environment.Get("P_IMAGE_NAME")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_IMAGE_NAME"))
	}
	if environment.IsIgnore() {
		return nil
	}
	username, password, err := dockerAuth()
	if err != nil {
		return err
	}
	docker, err := engine.NewAuthDockerCli(working, *username, *password)
	if err != nil {
		return err
	}
	err = docker.DockerPush(imageName)
	if err != nil {
		return err
	}
	return nil
}

// DockerPull 容器镜像拉取.
func DockerPull() error {
	// 读取工作目录
	working, ok := environment.Get("CI_PROJECT_DIR")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_PROJECT_DIR"))
	}
	// 读取镜像名称
	imageName, ok := environment.Get("P_IMAGE_NAME")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_IMAGE_NAME"))
	}
	if environment.IsIgnore() {
		return nil
	}
	username, password, err := dockerAuth()
	if err != nil {
		return err
	}
	docker, err := engine.NewAuthDockerCli(working, *username, *password)
	if err != nil {
		return err
	}
	err = docker.DockerPull(imageName)
	if err != nil {
		return err
	}
	return err
}

// DockerTag 容器镜像别名.
func DockerTag() error {
	// 读取工作目录
	working, ok := environment.Get("CI_PROJECT_DIR")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_PROJECT_DIR"))
	}
	if environment.IsIgnore() {
		return nil
	}
	username, password, err := dockerAuth()
	if err != nil {
		return err
	}
	scriptEngine, err := engine.NewAuthDockerCli(working, *username, *password)
	if err != nil {
		return err
	}
	// 读取镜像名称
	oldImageName, ok := environment.Get("oldImageName")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "oldImageName"))
	}
	newImageName, ok := environment.Get("newImageName")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "newImageName"))
	}
	err = scriptEngine.DockerTag(oldImageName, newImageName)
	if err != nil {
		return err
	}
	return nil
}

// DockerRemove 容器镜像删除.
func DockerRemove() error {
	// 读取工作目录
	working, ok := environment.Get("CI_PROJECT_DIR")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_PROJECT_DIR"))
	}
	if environment.IsIgnore() {
		return nil
	}
	// 读取镜像名称
	imageName, ok := environment.Get("imageName")
	if !ok {
		imageName, ok = environment.Get("P_IMAGE_NAME")
		if !ok {
			return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_IMAGE_NAME"))
		}
	}
	username, password, err := dockerAuth()
	if err != nil {
		return err
	}
	scriptEngine, err := engine.NewAuthDockerCli(working, *username, *password)
	if err != nil {
		return err
	}
	err = scriptEngine.DockerRemove(imageName)
	if err != nil {
		return err
	}
	return nil
}

// KubernetesRelease 发布服务.
func KubernetesRelease() error {
	// 读取集群名称
	colony, ok := environment.Get("P_COLONY")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_COLONY"))
	}
	colony = strings.ToUpper(colony)
	// 读取集群环境名称
	colonyEnv, ok := environment.Get("colonyEnv")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "colonyEnv"))
	}
	// 读取命名空间
	namespace, ok := environment.Get("P_NAMESPACE_" + strings.ToUpper(colonyEnv))
	if !ok {
		namespace, ok = environment.Get("P_NAMESPACE")
		if !ok {
			return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_NAMESPACE"))
		}
	}
	// 读取镜像名称
	imageName, ok := environment.Get("P_IMAGE_NAME")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_IMAGE_NAME"))
	}
	// 读取服务名称
	serviceName, ok := environment.Get("P_SERVICE_NAME")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_SERVICE_NAME"))
	}
	if environment.IsIgnore() {
		return nil
	}

	// 更新服务
	err := engine.ExecuteReleaseService(colony, colonyEnv, namespace, serviceName, imageName)
	if err != nil {
		return err
	}

	// 通知
	_ = SendMessage(true, "")

	return nil
}

// SSHRelease 发布服务.
func SSHRelease() error {
	// 读取工作目录
	working, ok := environment.Get("CI_PROJECT_DIR")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_PROJECT_DIR"))
	}
	// 读取服务器
	server, ok := environment.Get("server")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "server"))
	}
	// 读取服务器路径
	serverPath, ok := environment.Get("path")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "path"))
	}
	// 读取服务器需要执行的脚本
	before, ok := environment.Get("before")
	after, ok := environment.Get("after")
	// 更新服务
	err := engine.ExecuteSSHReleaseService(working, server, serverPath, before, after)
	if err != nil {
		return err
	}

	// 通知
	_ = SendMessage(true, "")
	return nil
}

// NacosSync 同步配置.
func NacosSync() error {
	// 服务类型
	serviceType, ok := environment.Get("P_SERVICE_TYPE")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_SERVICE_TYPE"))
	}
	// 服务实例ID
	instanceId, ok := environment.Get("P_INSTANCE_ID")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_INSTANCE_ID"))
	}
	// 命名空间
	instanceNamespace, ok := environment.Get("P_INSTANCE_NAMESPACE")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_INSTANCE_NAMESPACE"))
	}
	// 配置目录
	workDirectory, ok := environment.Get("CI_PROJECT_DIR")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_PROJECT_DIR"))
	}
	nacosDirectory, ok := environment.Get("P_CONFIG_DIRECTORY")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_CONFIG_DIRECTORY"))
	}

	if environment.IsIgnore() {
		return nil
	}

	if serviceType == "ALIYUN" {
		err := engine.AliyunNacosSync(instanceId, instanceNamespace, path.Join(workDirectory, nacosDirectory))
		if err != nil {
			return err
		}
	} else if serviceType == "TENCENT" {
		err := engine.TencentNacosSync(instanceId, instanceNamespace, path.Join(workDirectory, nacosDirectory))
		if err != nil {
			return err
		}
	}

	// 通知
	_ = SendMessage(true, "")

	return nil
}

// SendMessage 发送消息.
func SendMessage(status bool, content string) error {
	// 写入消息发送成功
	environment.Put("GO_RELEASE_RESULT", fmt.Sprint(status))
	environment.Put("GO_RELEASE_ERROR_MESSAGE", content)
	// 读取企业微信通知地址
	robotUrl, ok := environment.Get("GL_MESSAGE_CP_WECHAT_ROBOT")
	if ok {
		// 读取企业微信消息模板
		var messageTemplate string
		messageTemplate, ok = environment.Get("GL_MESSAGE_CP_WECHAT_TEMPLATE")
		if ok {
			color.Blue(fmt.Sprintf("[Message] 推送消息..."))
			var messageClient = message.NewCPWeChat(robotUrl)
			err := messageClient.Push("", messageTemplate)
			if err != nil {
				color.Yellow(fmt.Sprint(err))
				return err
			}
		}
	}
	return nil
}
