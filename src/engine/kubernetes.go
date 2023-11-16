package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/nuwa/bpp.v3/environment"
	"github.com/samber/lo"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
)

// 集群配置文件 环境变量前缀
const colonyKeyPrefix = "GS_RELEASE_KUBERNETES_"

// 镜像名称配置
const imageNameConfigKey = "GL_IMAGE_NAME_CONFIG"

// 镜像拉取策略 始终拉取最新
const imagePullPolicyAlways = "Always"

// Pod 标签选择器
const (
	deploymentDefaultLabelSelectKey = "app"        // 默认 Kubernetes 集群
	deploymentK8SLabelSelectKey     = "k8s-app"    // 阿里云 Kubernetes 集群
	deploymentTencentLabelSelectKey = "qcloud-app" // 腾讯云 Kubernetes 集群
)

type Kubernetes struct {
	*kubernetes.Clientset
	colony string // 集群名称
}

// NewConfigClient 基于内存文件创建 Kubernetes 客户端
func NewConfigClient(colony, env, configContent string) (*Kubernetes, error) {
	// 写出配置文件
	var configPath = colony + "-" + env + ".yaml"
	err := os.WriteFile(configPath, []byte(configContent), 644)
	if err != nil {
		return nil, err
	}
	return NewKubernetesClient(colony, configPath)
}

// NewKubernetesClient 创建 Kubernetes 客户端
func NewKubernetesClient(colony, configPath string) (*Kubernetes, error) {
	// 加载配置文件
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, err
	}
	restClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Kubernetes{Clientset: restClient, colony: colony}, nil
}

// ListNamespace 查询所有命名空间
func (k Kubernetes) ListNamespace() []v1.Namespace {
	list, err := k.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	return list.Items
}

// Deployments 查询所有的无状态服务
func (k Kubernetes) Deployments(namespace, deployment string) *v12.Deployment {
	result, err := k.AppsV1().Deployments(namespace).Get(context.Background(), deployment, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return result
}

// ListPod 查询符合选择器状态的Pod
func (k Kubernetes) ListPod(namespace, selector string) []v1.Pod {
	list, err := k.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil
	}
	return list.Items
}

// DeletePod 根据名称删除Pod
func (k Kubernetes) DeletePod(namespace, podName string) (bool, error) {
	color.Green(fmt.Sprintf("[Kubernetes] Delete %s Pods : %s", namespace, podName))
	err := k.CoreV1().Pods(namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// UpdateDeploymentImage 更新无状态服务的镜像版本
func (k Kubernetes) UpdateDeploymentImage(namespace, deploymentName, imageName string) (bool, error) {
	color.Green(fmt.Sprintf("[Kubernetes] Update Deployment Image : %s/%s <- %s", namespace, deploymentName, imageName))
	var item = map[string]interface{}{}
	item["op"] = "replace"
	item["path"] = "/spec/template/spec/containers/0/image"
	item["value"] = imageName
	requestByteData, err := json.Marshal([]map[string]interface{}{item})
	if err != nil {
		return false, err
	}
	_, err = k.AppsV1().Deployments(namespace).Patch(context.Background(), deploymentName,
		types.JSONPatchType, requestByteData, metav1.PatchOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// UpdateDeploymentImagePullPolicyAlways 更新无状态服务的镜像拉取策略为始终拉取最新
func (k Kubernetes) UpdateDeploymentImagePullPolicyAlways(namespace, deploymentName string) (bool, error) {
	var item = map[string]interface{}{}
	item["op"] = "replace"
	item["path"] = "/spec/template/spec/containers/0/imagePullPolicy"
	item["value"] = imagePullPolicyAlways
	requestByteData, err := json.Marshal([]map[string]interface{}{item})
	if err != nil {
		return false, err
	}
	_, err = k.AppsV1().Deployments(namespace).Patch(context.Background(), deploymentName,
		types.JSONPatchType, requestByteData, metav1.PatchOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// ReleaseService 发布服务
func (k Kubernetes) ReleaseService(namespace, serviceName, newImageName string) error {
	// 读取服务是否存在
	deploymentInfo := k.Deployments(namespace, serviceName)
	if deploymentInfo == nil {
		return errors.New(fmt.Sprintf("%s -> %s/%s Service Not", k.colony, namespace, serviceName))
	}
	// 检查服务配置是是总是拉取
	if deploymentInfo.Spec.Template.Spec.Containers[0].ImagePullPolicy != imagePullPolicyAlways {
		_, err := k.UpdateDeploymentImagePullPolicyAlways(namespace, serviceName)
		if err != nil {
			return err
		}
	}
	// 如果镜像相同则删除POD 否则 修改服务镜像版本
	color.Blue(fmt.Sprintf("Deployment ImageName: %s", deploymentInfo.Spec.Template.Spec.Containers[0].Image))
	if deploymentInfo.Spec.Template.Spec.Containers[0].Image == newImageName {
		color.Blue("Update Pods Image  ...")
		pods := k.ListPod(namespace, deploymentDefaultLabelSelectKey+"="+serviceName)
		// 官方
		if pods == nil || len(pods) == 0 {
			pods = k.ListPod(namespace, deploymentK8SLabelSelectKey+"="+serviceName)
		}
		// 腾讯云
		if pods == nil || len(pods) == 0 {
			pods = k.ListPod(namespace, deploymentTencentLabelSelectKey+"="+serviceName)
		}
		// 如果找不到 Pod
		if pods == nil || len(pods) == 0 {
			return errors.New(fmt.Sprintf("%s -> %s/%s Pod Not", k.colony, namespace, serviceName))
		}
		for _, pod := range pods {
			_, err := k.DeletePod(namespace, pod.Name)
			if err != nil {
				return err
			}
		}
	} else {
		color.Blue("Deployment Image  ...")
		_, err := k.UpdateDeploymentImage(namespace, serviceName, newImageName)
		if err != nil {
			return err
		}
	}
	return nil
}

// parseImageName 解析镜像名称
func parseImageName(colony, imageName string) (*string, error) {
	colony = strings.ToUpper(colony)
	if configValue, ok := environment.Get(imageNameConfigKey); ok {
		var config map[string]interface{}
		err := json.Unmarshal([]byte(configValue), &config)
		if err != nil {
			return nil, err
		}
		for key, value := range config {
			if strings.ToUpper(key) == colony {
				var values = strings.Split(fmt.Sprint(value), "->")
				if len(values) < 2 {
					continue
				}
				var newImageName = strings.ReplaceAll(imageName, values[0], values[1])
				return &newImageName, err
			}
		}
	}
	return &imageName, nil
}

// ExecuteReleaseService 执行发布服务.
func ExecuteReleaseService(colony, env, namespace, serviceName, imageName string) error {
	color.Blue(fmt.Sprintf("[Kubernetes] 集群: %s 环境: %s 命名空间: %s 服务名称: %s 镜像名称: %s", colony, env, namespace, serviceName, imageName))
	var actuator = func(f func(colony, namespace string) error) error {
		for _, c := range strings.Split(colony, ",") {
			for _, n := range strings.Split(namespace, ",") {
				var err = f(strings.ToUpper(c), n)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	return actuator(func(colony, namespace string) error {
		if colony == "" || namespace == "" {
			return errors.New("colony or namespace is empty")
		}

		kubernetesConfig, err := func() (*string, error) {
			value, ok := environment.Get(colonyKeyPrefix + colony + "_" + strings.ToUpper(env))
			if ok {
				return &value, nil
			}
			value, ok = environment.Get(colonyKeyPrefix + colony)
			if ok {
				return &value, nil
			}
			return nil, errors.New(fmt.Sprintf("Kubernetes colony Config (%s or %s) Find Not",
				colonyKeyPrefix+colony+"_"+strings.ToUpper(env), colonyKeyPrefix+colony))
		}()
		if err != nil {
			return err
		}

		// 解析镜像名称
		newImageName, err := parseImageName(colony, imageName)
		if err != nil {
			return err
		}
		color.Green(fmt.Sprintf("Release to Kubernetes (%s) %s / %s <-- %s",
			colony, serviceName, namespace, *newImageName))

		// 初始化客户端
		kubernetesClient, err := NewConfigClient(colony, env, *kubernetesConfig)
		if err != nil {
			return err
		}

		// 读取命名空间是否存在
		namespaces := kubernetesClient.ListNamespace()
		if namespaces == nil {
			color.Yellow(fmt.Sprintf("[Kubernetes] 集群: %s 命名空间为空", colony))
			return nil
		}
		_, namespaceExist := lo.Find(namespaces, func(item v1.Namespace) bool {
			return item.Name == namespace
		})
		if !namespaceExist {
			color.Yellow(fmt.Sprintf("[Kubernetes] %s (env:%s) -> %s Namespace Not, ready to Find: %s", colony, env, namespace, namespace+"-"+env))
			namespace = namespace + "-" + env
			_, namespaceExist = lo.Find(namespaces, func(item v1.Namespace) bool {
				return item.Name == namespace
			})
		}
		if !namespaceExist {
			color.Yellow(fmt.Sprintf("[Kubernetes] %s (env:%s) -> %s Namespace Not", colony, env, namespace))
			return nil
		}

		color.Blue(fmt.Sprintf("[Kubernetes] Colony Workspace (%s) -> %s ", colony, namespace))

		// 刷新服务
		err = kubernetesClient.ReleaseService(namespace, serviceName, *newImageName)
		if err != nil {
			return err
		}
		return nil
	})
}
