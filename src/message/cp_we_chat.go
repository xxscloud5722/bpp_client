package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/nuwa/bpp.v3/environment"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"net/http"
	"strings"
)

type CPWeChatMessage struct {
	URL string // 机器人地址
}

// Push 发送消息.
func (message *CPWeChatMessage) Push(title, content string) error {
	releaseResult, _ := environment.Get("GO_RELEASE_RESULT")
	releaseMessage, _ := environment.Get("GO_RELEASE_ERROR_MESSAGE")

	// 准备的环境变量
	link, ok := environment.Get("CI_JOB_URL")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_JOB_URL"))
	}
	projectName, ok := environment.Get("P_PROJECT_NAME")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_PROJECT_NAME"))
	}
	author, ok := environment.Get("CI_COMMIT_AUTHOR")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_COMMIT_AUTHOR"))
	}
	env, ok := environment.Get("P_COLONY_ENV")
	if !ok {
		env, ok = environment.Get("colonyEnv")
		if !ok {
			return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "P_COLONY_ENV"))
		}
	}
	commitRefName, ok := environment.Get("CI_COMMIT_REF_NAME")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_COMMIT_REF_NAME"))
	}
	description, ok := environment.Get("CI_COMMIT_MESSAGE")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_COMMIT_MESSAGE"))
	}
	commitId, ok := environment.Get("CI_COMMIT_SHA")
	if !ok {
		return errors.New(fmt.Sprintf("Environment variable ${%s} not exist", "CI_COMMIT_SHA"))
	}

	// 构建请求
	var args = map[string]string{}
	args["title"] = lo.If(releaseResult == "true", "<font color=\"info\">构建成功</font>").Else("<font color=\"warning\">构建失败</font>")
	args["failReason"] = lo.If(releaseResult == "true", "").Else("<br/>> **失败原因**：[点击查看失败详情](" + link + ")  " + releaseMessage)
	args["projectName"] = projectName
	args["author"] = author
	if strings.ToUpper(env) == "DEV" {
		args["env"] = "开发环境"
	} else if strings.ToUpper(env) == "TEST" {
		args["env"] = "测试环境"
	} else if strings.ToUpper(env) == "PREV" {
		args["env"] = "预发布环境"
	} else if strings.ToUpper(env) == "PROD" {
		args["env"] = "生产环境"
	}
	args["commitRefName"] = commitRefName
	args["description"] = description
	args["commitId"] = commitId
	args["link"] = link

	content, err := ParseTemplateParam(content, args)
	if err != nil {
		return err
	}
	color.Blue("[Message] 消息内容: ")
	color.Blue(content)

	var requestBody = map[string]interface{}{}
	var contentRequest = map[string]string{}
	contentRequest["content"] = content
	requestBody["msgtype"] = "markdown"
	requestBody["markdown"] = contentRequest
	requestByteData, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", message.URL, bytes.NewReader(requestByteData))
	if err != nil {
		return err
	}
	var client = http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode == 200 {
		return nil
	}
	return errors.New(fmt.Sprintf("weChat Response Error Code: %d", response.StatusCode))
}
