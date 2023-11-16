package console

import (
	"errors"
	"fmt"
	"github.com/nuwa/bpp.v3/engine"
	"github.com/nuwa/bpp.v3/environment"
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

	// 更新服务
	colonyEnv = strings.ToLower(colonyEnv)
	err := engine.ExecuteReleaseService(colony, colonyEnv, namespace, serviceName, imageName)
	if err != nil {
		return err
	}

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

	return nil
}
