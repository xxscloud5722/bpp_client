package engine

import (
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	mse "github.com/alibabacloud-go/mse-20190531/v3/client"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/nuwa/bpp.v3/environment"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"encoding/json"
)

type AliyunNacos struct {
	accessKeyId     string      // 资源ID.
	accessKeySecret string      // 资源密钥.
	instanceId      string      // 实例ID.
	client          *mse.Client // 实例.
}

func recursion(handle func() (bool, error)) error {
	for {
		result, err := handle()
		if err != nil {
			return err
		}
		if !result {
			break
		}
	}
	return nil
}

// NewAliyunNacos 创建阿里云Nacos 客户端.
func NewAliyunNacos(accessKeyId string, accessKeySecret string, instanceId string) (*AliyunNacos, error) {
	client, err := mse.NewClient(&openapi.Config{
		AccessKeyId:     &accessKeyId,
		AccessKeySecret: &accessKeySecret,
		Endpoint:        tea.String("mse.cn-shanghai.aliyuncs.com"),
	})
	if err != nil {
		return nil, err
	}
	return &AliyunNacos{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
		instanceId:      instanceId,
		client:          client,
	}, nil
}

// GetNacosConfigList 获取Nacos配置列表.
func (aliyun *AliyunNacos) GetNacosConfigList(namespaceId string) ([]mse.ListNacosConfigsResponseBodyConfigurations, error) {
	response := make([]mse.ListNacosConfigsResponseBodyConfigurations, 0)
	err := recursion(func() (bool, error) {
		listNacosConfigsRequest := &mse.ListNacosConfigsRequest{
			InstanceId:  tea.String(aliyun.instanceId),
			PageNum:     tea.Int32(1),
			PageSize:    tea.Int32(200),
			NamespaceId: tea.String(namespaceId),
		}
		result, err := aliyun.client.ListNacosConfigsWithOptions(listNacosConfigsRequest, &util.RuntimeOptions{})
		if err != nil {
			return false, err
		}
		for i := range result.Body.Configurations {
			response = append(response, *result.Body.Configurations[i])
		}
		return (*result.Body.PageNumber)*(*result.Body.TotalCount) < *result.Body.TotalCount, nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetNacosConfig 获取Nacos配置详情.
func (aliyun *AliyunNacos) GetNacosConfig(namespaceId string, group string, dataId string) (*mse.GetNacosConfigResponseBodyConfiguration, error) {
	request := &mse.GetNacosConfigRequest{
		InstanceId:  tea.String(aliyun.instanceId),
		NamespaceId: tea.String(namespaceId),
		Group:       tea.String(group),
		DataId:      tea.String(dataId),
	}
	result, err := aliyun.client.GetNacosConfigWithOptions(request, &util.RuntimeOptions{})
	if err != nil {
		return nil, err
	}
	return result.Body.Configuration, nil
}

// DeleteNacosConfig 删除Nacos 配置.
func (aliyun *AliyunNacos) DeleteNacosConfig(namespaceId string, group string, dataId string) (*bool, error) {
	request := &mse.DeleteNacosConfigRequest{
		InstanceId:  tea.String(aliyun.instanceId),
		NamespaceId: tea.String(namespaceId),
		Group:       tea.String(group),
		DataId:      tea.String(dataId),
	}
	result, err := aliyun.client.DeleteNacosConfigWithOptions(request, &util.RuntimeOptions{})
	if err != nil {
		return nil, err
	}
	return result.Body.Success, nil
}

// UpdateNacosConfig 修改Nacos 配置.
func (aliyun *AliyunNacos) UpdateNacosConfig(namespaceId string, group string, dataId string, content *string, fileType string) (*bool, error) {
	request := &mse.UpdateNacosConfigRequest{
		InstanceId:  tea.String(aliyun.instanceId),
		NamespaceId: tea.String(namespaceId),
		Group:       tea.String(group),
		DataId:      tea.String(dataId),
		Content:     content,
		Type:        tea.String(strings.TrimPrefix(fileType, ".")),
	}
	result, err := aliyun.client.UpdateNacosConfigWithOptions(request, &util.RuntimeOptions{})
	if err != nil {
		return nil, err
	}
	return result.Body.Success, nil
}

// CreateNacosConfig 创建Nacos 配置.
func (aliyun *AliyunNacos) CreateNacosConfig(namespaceId string, group string, dataId string, content *string, fileType string) (*bool, error) {
	request := &mse.CreateNacosConfigRequest{
		InstanceId:  tea.String(aliyun.instanceId),
		NamespaceId: tea.String(namespaceId),
		Group:       tea.String(group),
		DataId:      tea.String(dataId),
		Content:     content,
		Type:        tea.String(fileType),
	}
	result, err := aliyun.client.CreateNacosConfigWithOptions(
		request, &util.RuntimeOptions{})
	if err != nil {
		return nil, err
	}
	return result.Body.Success, nil
}

// PullNacos 拉取配置到本地磁盘.
func (aliyun *AliyunNacos) PullNacos(namespaceId string, rootPath string) error {
	if _, err := os.Stat(rootPath); err != nil || os.IsNotExist(err) {
		// 创建目录
		err := os.MkdirAll(rootPath, 644)
		if err != nil {
			return err
		}
	}
	configList, err := aliyun.GetNacosConfigList(namespaceId)
	if err != nil {
		return err
	}
	for i := range configList {
		var item = configList[i]
		log.Println(">>", namespaceId, "/", *item.DataId)
		config, err := aliyun.GetNacosConfig(namespaceId, *item.Group, *item.DataId)
		if err != nil {
			return err
		}
		var fileName string
		if *config.Type == "" {
			fileName = *config.DataId
		} else {
			fileName = *config.DataId + "." + *config.Type
		}
		fileWrite, err := os.Create(path.Join(rootPath, fileName))
		if err != nil {
			return err
		}
		_, err = io.WriteString(fileWrite, *config.Content)
		if err != nil {
			return err
		}
	}
	return nil
}

// Sync 同步配置到线上.
func (aliyun *AliyunNacos) Sync(namespaceId string, rootPath string) (*bool, error) {
	log.Println(fmt.Sprintf("Sync Nacos Config By NamespaceId: %s", namespaceId))
	// 读取本地磁盘
	var files []string
	var pathSeparator = string(os.PathSeparator)
	rootPath = strings.ReplaceAll(rootPath, "\\", pathSeparator)
	rootPath = strings.ReplaceAll(rootPath, "/", pathSeparator)
	err := filepath.Walk(rootPath, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		var relativePath = strings.TrimPrefix(path, rootPath)
		files = append(files, strings.TrimPrefix(strings.TrimPrefix(relativePath, "/"), "\\"))
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 读取线上配置
	configList, err := aliyun.GetNacosConfigList(namespaceId)
	if err != nil {
		return nil, err
	}

	// 是否存在新增
	var addTask []string
	var addTaskConfig []string
	for i := range files {
		_, result := lo.Find(configList, func(item mse.ListNacosConfigsResponseBodyConfigurations) bool {
			return strings.TrimSuffix(files[i], path.Ext(files[i])) == *item.DataId
		})
		if !result {
			addTask = append(addTask, files[i])
			fileByte, err := os.ReadFile(path.Join(rootPath, files[i]))
			if err != nil {
				return nil, err
			}
			addTaskConfig = append(addTaskConfig, string(fileByte))
		}
	}

	// 是否存在修改
	var updateTask []string
	var updateConfig []string
	var updateConfigId []string
	for i := range configList {
		filePath, result := lo.Find(files, func(item string) bool {
			return strings.TrimSuffix(item, path.Ext(item)) == *configList[i].DataId
		})
		if !result {
			continue
		}
		var item = configList[i]
		config, err := aliyun.GetNacosConfig(namespaceId, *item.Group, *item.DataId)
		if err != nil {
			return nil, err
		}
		fileByte, err := os.ReadFile(path.Join(rootPath, filePath))
		if err != nil {
			return nil, err
		}
		var fileContent = string(fileByte)
		if *config.Content == fileContent && *config.Type == strings.TrimPrefix(path.Ext(filePath), ".") {
			continue
		}
		updateTask = append(updateTask, filePath)
		updateConfig = append(updateConfig, fileContent)
		updateConfigId = append(updateConfigId, *config.DataId)
	}

	// 是否存在删除
	var deleteTask []mse.ListNacosConfigsResponseBodyConfigurations
	for i := range configList {
		_, result := lo.Find(files, func(item string) bool {
			return strings.TrimSuffix(item, path.Ext(item)) == *configList[i].DataId
		})
		if !result {
			deleteTask = append(deleteTask, configList[i])
		}
	}

	log.Println(fmt.Sprintf("Create: %d条", len(addTask)))
	lo.ForEach(addTask, func(it string, i int) {
		_, err := aliyun.CreateNacosConfig(namespaceId, "DEFAULT_GROUP",
			strings.TrimSuffix(it, path.Ext(it)), &addTaskConfig[i], strings.TrimPrefix(path.Ext(it), "."))
		if err != nil {
			log.Fatalf("Error:%s", err)
		}
	})
	log.Println(fmt.Sprintf("Update: %d条", len(updateTask)))
	lo.ForEach(updateTask, func(it string, i int) {
		_, err := aliyun.UpdateNacosConfig(namespaceId, "DEFAULT_GROUP", updateConfigId[i], &updateConfig[i], path.Ext(it))
		if err != nil {
			log.Fatalf("Error:%s", err)
		}
	})
	log.Println(fmt.Sprintf("Delete: %d条", len(deleteTask)))
	lo.ForEach(deleteTask, func(it mse.ListNacosConfigsResponseBodyConfigurations, _ int) {
		_, err := aliyun.DeleteNacosConfig(namespaceId, *it.Group, *it.DataId)
		if err != nil {
			log.Fatalf("Error:%s", err)
		}
	})
	return tea.Bool(true), nil
}

func AliyunNacosSync(instanceKey, instanceNamespace, nacosDirectory string) error {
	var key = configNacosKey + instanceKey
	instanceJson, ok := environment.Get(key)
	if !ok {
		return errors.New(fmt.Sprintf("Run Sync Nacos Not Find Param: %s", key))
	}
	instanceConfigMap := make(map[string]string)
	err := json.Unmarshal([]byte(instanceJson), &instanceConfigMap)
	if err != nil {
		return err
	}
	accessKeyId, accessKeyIdOk := instanceConfigMap["accessKeyId"]
	accessKeySecret, accessKeySecretOk := instanceConfigMap["accessKeySecret"]
	instanceId, instanceIdOk := instanceConfigMap["instanceId"]
	if !accessKeyIdOk || !accessKeySecretOk || !instanceIdOk {
		return errors.New("Aliyun Config Json Param Error")
	}
	aliyun, err := NewAliyunNacos(accessKeyId, accessKeySecret, instanceId)
	if err != nil {
		return err
	}
	_, err = aliyun.Sync(instanceNamespace, nacosDirectory)
	if err != nil {
		return err
	}
	return nil
}
