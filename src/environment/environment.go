package environment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/nuwa/bpp.v3/common"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var environmentMap = map[string]string{}
var serverTokenKey = "GL_SERVER_ACCESS_TOKEN"
var serverUrlKey = "GL_SERVER_URL"
var client = &http.Client{}

// init 自动加载程序参数变量.
func init() {
	// 默认加载运行时参数
	for _, item := range os.Args {
		var values = strings.Split(item, "=")
		if len(values) <= 1 {
			continue
		}
		if values[0] == "" || values[1] == "" {
			continue
		}
		color.Blue(fmt.Sprintf("[Environment] Load Key : %s: %s", values[0], values[1]))
		environmentMap[values[0]] = values[1]
	}

	// 默认加载全局环境变量
	for _, env := range os.Environ() {
		var index = strings.Index(env, "=")
		if index >= 0 {
			environmentMap[env[0:index]] = env[index+1:]
		}
	}

	// 默认加载GL开头全局参数
	result, err := GetGL()
	if err != nil {
		return
	}
	for _, item := range result {
		environmentMap[item[0]] = item[1]
	}
}

// LocalMap 返回本地Map.
func LocalMap() map[string]string {
	return environmentMap
}

// Put 添加元素到本地.
func Put(key, value string) {
	environmentMap[key] = value
}

// getServer 获取服务器信息
func getServer() (*string, *string, error) {
	// 读取令牌
	var accessToken string
	if value, ok := environmentMap[serverTokenKey]; ok {
		accessToken = value
	} else {
		accessToken = os.Getenv(serverTokenKey)
	}
	// 读取地址
	var url string
	if value, ok := environmentMap[serverUrlKey]; ok {
		url = value
	} else {
		url = os.Getenv(serverUrlKey)
	}
	if url == "" {
		url = "http://127.0.0.1:8080"
	}
	return &url, &accessToken, nil
}

// post 请求.
func post(path string, data map[string]interface{}) ([]byte, error) {
	url, accessToken, err := getServer()
	if err != nil {
		return nil, err
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", fmt.Sprintf("%s%s?access-token=%s", *url, path, *accessToken), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	return io.ReadAll(response.Body)
}

// get 请求.
func get(path string) ([]byte, error) {
	url, accessToken, err := getServer()
	if err != nil {
		return nil, err
	}
	response, err := http.Get(fmt.Sprintf("%s%s?access-token=%s", *url, path, *accessToken))
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)

	return io.ReadAll(response.Body)
}

// getByServer 获取KeyValue 服务的参数.
func getByServer(key string) (string, bool) {
	response, err := get("/pair/" + key)
	if err != nil {
		return "", false
	}
	var result struct {
		Success bool   `json:"success"` // 是否响应成功
		Data    string `json:"data"`    // 响应数据
	}
	err = json.Unmarshal(response, &result)
	if err != nil || !result.Success {
		return "", false
	}
	return result.Data, true
}

// Get 必须获取环境变量.
func Get(key string) (string, bool) {
	// 读取内存变量
	if value, ok := environmentMap[key]; ok && value != "" {
		return value, true
	}
	// 读取服务器变量
	if value, ok := getByServer(key); ok && value != "" {
		return value, true
	}
	return "", false
}

// GetGL 获取全局GL开头的环境变量.
func GetGL() ([][]string, error) {
	response, err := get("/pair/list/GL")
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"data"`
	}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, err
	}
	var rows [][]string
	for _, item := range result.Data {
		rows = append(rows, []string{item.Key, item.Value})
	}
	return rows, nil
}

// Print 打印全部环境变量.
func Print(prefix string) error {
	color.Green("Get All ...")
	var response []byte
	var err error
	if prefix == "" {
		response, err = get("/pair/list")
	} else {
		response, err = get("/pair/list/" + prefix)
	}
	if err != nil {
		return err
	}
	var result struct {
		Success bool `json:"success"`
		Data    []struct {
			Key         string `json:"key,omitempty"`
			Value       string `json:"value,omitempty"`
			Description string `json:"description,omitempty"`
		} `json:"data"`
	}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return err
	}
	var table [][]string
	for index, item := range result.Data {
		table = append(table, []string{strconv.Itoa(index + 1), item.Key, valueParse(item.Value), item.Description})
	}
	common.PrintTable([]string{"序号", "Key", "Value", "描述"}, table)
	return nil
}

// PrintByKey 打印指定环境变量.
func PrintByKey(key string) error {
	color.Green(fmt.Sprintf("Get Key: %s ...", key))
	response, err := get("/pair/" + key)
	if err != nil {
		return err
	}
	var result struct {
		Success bool   `json:"success"`
		Data    string `json:"data"`
	}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return err
	}
	if !result.Success || result.Data == "" {
		return errors.New("Key: " + key + " Not Value")
	}
	// 打印输出
	color.Blue(result.Data)
	return nil
}

// valueParse 打印显示序列化.
func valueParse(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	var rows = strings.Split(value, "\n")
	value = rows[0]
	value = strings.ReplaceAll(value, " ", "　")
	value = strings.ReplaceAll(value, "|", "")
	value = strings.ReplaceAll(value, "-", "")
	if len(value) > 100 {
		return value[0:100] + " ..."
	} else if len(rows) > 1 {
		return value + " ..."
	}
	return value
}

// Push 添加环境变量到服务器.
func Push(key, value, description string) error {
	if strings.HasPrefix(value, "#file://") {
		file, err := os.ReadFile(value[8:])
		if err != nil {
			return err
		}
		value = string(file)
	}
	data := map[string]interface{}{
		"key":         key,
		"value":       value,
		"description": description,
	}
	response, err := post("/pair/save", data)
	if err != nil {
		return err
	}
	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return err
	}
	if !result.Success {
		return errors.New(result.Message)
	}
	return nil
}

// Remove 删除环境变量.
func Remove(key string) error {
	data := map[string]interface{}{
		"key": key,
	}
	response, err := post("/pair/remove", data)
	if err != nil {
		return err
	}
	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return err
	}
	if !result.Success {
		return errors.New(result.Message)
	}
	return nil
}

// IsIgnore 是否忽略执行.
func IsIgnore() bool {
	stage, ok := Get("CI_JOB_STAGE")
	if !ok {
		return false
	}
	message, ok := Get("CI_COMMIT_MESSAGE")
	if !ok {
		return false
	}
	stage = strings.TrimSpace(strings.ToUpper(stage))
	message = strings.ToUpper(message)
	// 约定消息是C.开头的则根据执行的stage 执行相关代码
	if strings.HasPrefix(message, "C.") {
		return !strings.HasPrefix(message, "C."+strings.ToUpper(stage))
	}
	return false
}
