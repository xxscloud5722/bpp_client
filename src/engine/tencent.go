package engine

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/nuwa/bpp.v3/environment"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type TencentNacos struct {
	host        string // 域名.
	username    string // Nacos 用户名称.
	password    string // Nacos 用户密码.
	accessToken string // 访问令牌.
}

type TencentNacosNamespace struct {
	Namespace         string `json:"namespace"`         // Nacos 命名空间ID.
	NamespaceShowName string `json:"namespaceShowName"` //  Nacos 命名空间名称.
	ConfigCount       int    `json:"configCount"`       //  Nacos 命名空间配置数量.
}

type TencentNacosConfigItem struct {
	Namespace string `json:"tenant"`  // 命名空间ID.
	Id        string `json:"id"`      // ID.
	DataId    string `json:"dataId"`  // 数据ID.
	Group     string `json:"group"`   // 分组ID (默认 DEFAULT_GROUP).
	Content   string `json:"content"` // 配置文件内容.
	Md5       string `json:"md5"`     // 配置文件签名.
	Type      string `json:"type"`    // 配置文件类型.
}

// URL 获取腾讯云Nacos 地址.
func (tencent *TencentNacos) URL() string {
	if strings.HasPrefix(tencent.host, "http") {
		return tencent.host
	}
	return "http://" + tencent.host + ":8080"
}

// Post 腾讯云Nacos发布请求.
func (tencent *TencentNacos) Post(path string, form url.Values, v any) error {
	response, err := http.PostForm(tencent.URL()+path, form)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if v == nil {
		if string(bodyBytes) == "true" {
			return nil
		}
		return errors.New("Response Fail")
	}
	err = json.Unmarshal(bodyBytes, v)
	if err != nil {
		return err
	}
	return nil
}

// Get 腾讯云Nacos发布请求.
func (tencent *TencentNacos) Get(path string, v any) error {
	response, err := http.Get(tencent.URL() + path)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bodyBytes, v)
	if err != nil {
		return err
	}
	return nil
}

// Delete 腾讯云Nacos发布请求.
func (tencent *TencentNacos) Delete(path string, form url.Values) error {
	client := http.Client{}
	request, err := http.NewRequest(http.MethodDelete, tencent.URL()+path, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if string(bodyBytes) == "true" {
		return nil
	}
	return errors.New("Response Fail")
}

// NewTencent 通过URL 用户名以及密码创建实例
func NewTencent(host string, username string, password string) (*TencentNacos, error) {
	var tencent = &TencentNacos{
		host:        host,
		username:    username,
		password:    password,
		accessToken: "",
	}
	var loginResponse struct {
		AccessToken string `json:"accessToken"`
	}
	err := tencent.Post("/nacos/v1/auth/users/login", url.Values{
		"username": {tencent.username},
		"password": {tencent.password},
	}, &loginResponse)
	if err != nil {
		return nil, err
	}
	tencent.accessToken = loginResponse.AccessToken
	return tencent, nil
}

// GetNacosConfigList 读取Nacos 配置
func (tencent *TencentNacos) GetNacosConfigList(namespaceId string) (*[]TencentNacosConfigItem, error) {
	var urlPath = fmt.Sprintf("/nacos/v1/cs/configs?dataId=&group=&appName=&config_tags=&pageNo=1&pageSize=300&tenant=%s&search=accurate&accessToken=%s&username=nacos", namespaceId, tencent.accessToken)
	var response struct {
		TotalCount int                      `json:"totalCount"`
		PageNumber int                      `json:"pageNumber"`
		PageItems  []TencentNacosConfigItem `json:"pageItems"`
	}
	err := tencent.Get(urlPath, &response)
	if err != nil {
		return nil, err
	}
	return &response.PageItems, nil
}

// GetNacosConfig 获取Nacos配置详情.
func (tencent *TencentNacos) GetNacosConfig(namespaceId string, group string, dataId string) (*TencentNacosConfigItem, error) {
	var urlPath = fmt.Sprintf("/nacos/v1/cs/configs?show=all&dataId=%s&group=%s&tenant=%s&accessToken=%s&namespaceId=%s",
		dataId, group, namespaceId, tencent.accessToken, namespaceId)
	var response TencentNacosConfigItem
	err := tencent.Get(urlPath, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// DeleteNacosConfig 删除Nacos 配置.
func (tencent *TencentNacos) DeleteNacosConfig(namespaceId string, group string, dataId string) error {
	var urlPath = fmt.Sprintf("/nacos/v1/cs/configs?accessToken=%s&tenant=%s&group=%s&dataId=%s",
		tencent.accessToken, namespaceId, group, dataId)
	err := tencent.Delete(urlPath, url.Values{
		"namespaceId": {namespaceId},
	})
	if err != nil {
		return err
	}
	return nil
}

// UpdateNacosConfig 修改Nacos 配置.
func (tencent *TencentNacos) UpdateNacosConfig(namespaceId string, group string, dataId string, content string, fileType string) error {
	// Get
	nacosConfigItem, err := tencent.GetNacosConfig(namespaceId, group, dataId)
	if err != nil {
		return err
	}
	// 当前时间戳
	milliseconds := time.Now().UnixNano() / int64(time.Millisecond)
	// 签名
	hash := md5.Sum([]byte(content))
	// 将哈希值转换为十六进制字符串
	md5Hex := hex.EncodeToString(hash[:])
	var urlPath = fmt.Sprintf("/nacos/v1/cs/configs?accessToken=%s", tencent.accessToken)
	err = tencent.Post(urlPath, url.Values{
		"id":          {nacosConfigItem.Id},
		"md5":         {md5Hex},
		"createTime":  {strconv.FormatInt(milliseconds, 10)},
		"modifyTime":  {strconv.FormatInt(milliseconds, 10)},
		"dataId":      {dataId},
		"group":       {group},
		"content":     {content},
		"type":        {strings.TrimPrefix(fileType, ".")},
		"appName":     {},
		"tenant":      {namespaceId},
		"namespaceId": {namespaceId},
	}, nil)
	if err != nil {
		return err
	}
	return nil
}

// CreateNacosConfig 创建Nacos 配置.
func (tencent *TencentNacos) CreateNacosConfig(namespaceId string, group string, dataId string, content string, fileType string) error {
	var urlPath = fmt.Sprintf("/nacos/v1/cs/configs?accessToken=%s", tencent.accessToken)
	err := tencent.Post(urlPath, url.Values{
		"dataId":      {dataId},
		"group":       {group},
		"content":     {content},
		"type":        {fileType},
		"appName":     {},
		"tenant":      {namespaceId},
		"namespaceId": {namespaceId},
	}, nil)
	if err != nil {
		return err
	}
	return nil
}

// Sync 同步标签.
func (tencent *TencentNacos) Sync(namespaceId string, rootPath string) error {
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
		return err
	}
	// 读取线上配置
	configList, err := tencent.GetNacosConfigList(namespaceId)
	if err != nil {
		return err
	}

	// 是否存在新增
	var addTask []string
	var addTaskConfig []string
	for i := range files {
		_, result := lo.Find(*configList, func(item TencentNacosConfigItem) bool {
			return strings.TrimSuffix(files[i], path.Ext(files[i])) == item.DataId
		})
		if !result {
			addTask = append(addTask, files[i])
			fileByte, err := os.ReadFile(path.Join(rootPath, files[i]))
			if err != nil {
				return err
			}
			addTaskConfig = append(addTaskConfig, string(fileByte))
		}
	}

	// 是否存在修改
	var updateTask []string
	var updateConfig []string
	var updateConfigId []string
	for i := range *configList {
		filePath, result := lo.Find(files, func(item string) bool {
			return strings.TrimSuffix(item, path.Ext(item)) == (*configList)[i].DataId
		})
		if !result {
			continue
		}
		var item = (*configList)[i]
		config, err := tencent.GetNacosConfig(namespaceId, item.Group, item.DataId)
		if err != nil {
			return err
		}
		fileByte, err := os.ReadFile(path.Join(rootPath, filePath))
		if err != nil {
			return err
		}
		var fileContent = string(fileByte)
		if config.Content == fileContent && config.Type == strings.TrimPrefix(path.Ext(filePath), ".") {
			continue
		}
		updateTask = append(updateTask, filePath)
		updateConfig = append(updateConfig, fileContent)
		updateConfigId = append(updateConfigId, config.DataId)
	}

	// 是否存在删除
	var deleteTask []TencentNacosConfigItem
	for i := range *configList {
		_, result := lo.Find(files, func(item string) bool {
			return strings.TrimSuffix(item, path.Ext(item)) == (*configList)[i].DataId
		})
		if !result {
			deleteTask = append(deleteTask, (*configList)[i])
		}
	}

	log.Println(fmt.Sprintf("Create: %d条", len(addTask)))
	lo.ForEach(addTask, func(it string, i int) {
		err = tencent.CreateNacosConfig(namespaceId, "DEFAULT_GROUP",
			strings.TrimSuffix(it, path.Ext(it)), addTaskConfig[i], strings.TrimPrefix(path.Ext(it), "."))
		if err != nil {
			log.Fatalf("Error:%s", err)
		}
	})
	log.Println(fmt.Sprintf("Update: %d条", len(updateTask)))
	lo.ForEach(updateTask, func(it string, i int) {
		err := tencent.UpdateNacosConfig(namespaceId, "DEFAULT_GROUP", updateConfigId[i], updateConfig[i], path.Ext(it))
		if err != nil {
			log.Fatalf("Error:%s", err)
		}
	})
	log.Println(fmt.Sprintf("Delete: %d条", len(deleteTask)))
	lo.ForEach(deleteTask, func(it TencentNacosConfigItem, _ int) {
		err := tencent.DeleteNacosConfig(namespaceId, it.Group, it.DataId)
		if err != nil {
			log.Fatalf("Error:%s", err)
		}
	})
	return nil
}

// TencentNacosSync 腾讯云Nacos 同步.
func TencentNacosSync(instanceKey, instanceNamespace, nacosDirectory string) error {
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
	host, hostOk := instanceConfigMap["host"]
	username, usernameOk := instanceConfigMap["username"]
	password, passwordOk := instanceConfigMap["password"]
	if !hostOk || !usernameOk || !passwordOk {
		return errors.New("Tencent Config Json Param Error")
	}
	tencent, err := NewTencent(host, username, password)
	if err != nil {
		return err
	}
	err = tencent.Sync(instanceNamespace, nacosDirectory)
	if err != nil {
		return err
	}
	return nil
}
