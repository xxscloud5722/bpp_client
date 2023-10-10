package engine

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/nuwa/bpp.v3/environment"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

// 执行构建脚本的环境变量前缀
const scriptKey = "GL_BUILD_SCRIPT_"

// 配置文件的环境变量前缀
const configKey = "GL_BUILD_CONFIG_"

// 镜像构建文件环境变量前缀
const dockerfileKey = "GL_BUILD_DOCKERFILE_"

// 脚本输出文件名
const outputScriptFileName = "build.sh"

type ScriptEngine struct {
	working string // 工作目录
}

// NewScriptEngine 创建一个脚本生成引擎.
func NewScriptEngine(working string) *ScriptEngine {
	return &ScriptEngine{working: working}
}

// CreateBuild 创建一个构建脚本
func (engine *ScriptEngine) CreateBuild(projectId, packageType string) error {
	// 读取镜像类型
	imageName, ok := environment.Get("P_IMAGE_NAME")
	imageName = strings.TrimSpace(imageName)
	var content = []string{
		fmt.Sprintf("mkdir -p /opt/repository/%s", projectId),
	}
	// 如果有自己的定义的脚本
	scriptValue, ok := environment.Get(scriptKey + packageType)
	if ok {
		content = append(content, scriptValue)
	}
	var script string
	if !environment.IsIgnore() && ok && imageName != "" {
		content = append(content, "docker build -t "+imageName+" .")
	}
	script = engine.parse(strings.Join(content, "\n"))
	err := os.WriteFile(path.Join(engine.working, outputScriptFileName), []byte(script), 777)
	if err != nil {
		return err
	}
	color.Green("================ Deploy Script =================")
	color.Green(script)
	color.Green("================================================")
	return nil
}

// CreateConfigFile 如果类型无法匹配则匹配公共配置.
func (engine *ScriptEngine) CreateConfigFile(packageType string) error {
	// 创建匹配的配置文件
	err, result := engine.createConfigFile(packageType)
	if err != nil {
		return err
	}

	// 如果匹配成功, 则立即返回
	if result {
		return nil
	}

	// 创建公共配置文件
	err = engine.createCommonConfigFile(packageType)
	if err != nil {
		return err
	}

	return nil
}

// createConfigFile 创建一个指定配置文件.
func (engine *ScriptEngine) createConfigFile(packageType string) (error, bool) {
	var configType = strings.ToUpper(packageType)
	configValue, ok := environment.Get(configKey + configType)
	if !ok {
		return nil, false
	}
	var index = strings.Index(configValue, "#")
	if index < 0 {
		return errors.New(fmt.Sprintf("Run Create Build Config File Error: %s - %s", configKey+configType, configType)), false
	}
	var fileName = configValue[0:index]
	var config = engine.parse(configValue[index+1:])
	color.Green("================ " + fileName + " =================")
	color.Green(config)
	color.Green("================================================")
	err := os.WriteFile(path.Join(engine.working, fileName), []byte(config), 644)
	if err != nil {
		return err, false
	}
	return nil, true
}

// createCommonConfigFile 创建一个公共配置文件 Vue React Web 前缀是前端项目, MVN 前缀是后端项目.
func (engine *ScriptEngine) createCommonConfigFile(packageType string) error {
	var configType string
	var packageTypeUpper = strings.ToUpper(packageType)
	// 如果是前端项目
	if strings.HasPrefix(packageTypeUpper, "VUE") || strings.HasPrefix(packageTypeUpper, "REACT") ||
		strings.HasPrefix(packageTypeUpper, "WEB_") {
		configType = "NGINX"
	}
	// 如果是后端项目
	if strings.HasPrefix(packageTypeUpper, "MVN") {
		configType = "JAVA"
	}
	// 如果没有
	if configType == "" {
		return nil
	}
	configValue, ok := environment.Get(configKey + configType)
	if !ok {
		return errors.New(fmt.Sprintf("Run Create Build Config Not Find Param: %s", configKey+configType))
	}
	var index = strings.Index(configValue, "#")
	if index < 0 {
		return errors.New(fmt.Sprintf("Run Create Build Config File Error: %s - %s", configKey+configType, configType))
	}
	var fileName = configValue[0:index]
	var config = engine.parse(configValue[index+1:])
	color.Green("================ " + fileName + " =================")
	color.Green(config)
	color.Green("================================================")
	err := os.WriteFile(path.Join(engine.working, fileName), []byte(config), 644)
	if err != nil {
		return err
	}
	return nil
}

// CreateDockerFile 创建一个容器构建清单文件
func (engine *ScriptEngine) CreateDockerFile(dockerfileType string) error {
	dockerfileConfigValue, ok := environment.Get(dockerfileKey + dockerfileType)
	if !ok {
		return errors.New(fmt.Sprintf("Run Create Build Dockerfile Not Find Param: %s", dockerfileKey+dockerfileType))
	}
	var dockerfile = engine.parse(dockerfileConfigValue)
	color.Green("================ Dockerfile =================")
	color.Green(dockerfile)
	color.Green("================================================")
	err := os.WriteFile(path.Join(engine.working, "Dockerfile"), []byte(dockerfile), 644)
	if err != nil {
		return err
	}
	return nil
}

// parse 替换环境变量 `@#` 转换成 `$` 符号
func (engine *ScriptEngine) parse(content string) string {
	// 前端路径: 路径转换 (兼容)
	if value, ok := environment.Get("localPath"); ok && value != "" {
		content = strings.ReplaceAll(content, "#{localWebPath}", value)
		if value != "" {
			var paths = []string{"app"}
			paths = append(paths, strings.Split(value, "/")...)
			paths = paths[:len(paths)-1]
			content = strings.ReplaceAll(content, "#{localRootPath}", strings.Join(paths, "/"))
			content = strings.ReplaceAll(content, "#{webPath}", "/"+value)
		} else {
			content = strings.ReplaceAll(content, "#{localRootPath}", "app")
			content = strings.ReplaceAll(content, "#{webPath}", "")
		}
	} else {
		content = strings.ReplaceAll(content, "#{localWebPath}", "")
		content = strings.ReplaceAll(content, "#{localRootPath}", "")
		content = strings.ReplaceAll(content, "#{webPath}", "")
	}

	// 公共: 替换工作目录
	content = strings.ReplaceAll(content, "#{working}", engine.working)
	// 公共: 项目ID
	projectId, ok := environment.Get("CI_PROJECT_ID")
	if ok {
		content = strings.ReplaceAll(content, "#{id}", projectId)
	}
	// 公共: 项目名称
	name, ok := environment.Get("P_SERVICE_NAME")
	if ok {
		content = strings.ReplaceAll(content, "#{name}", name)
	}
	// 公共: 提交ID
	commitId, ok := environment.Get("CI_COMMIT_SHA")
	if ok {
		content = strings.ReplaceAll(content, "#{commitId}", commitId)
	}
	// 公共: 构建时间
	content = strings.ReplaceAll(content, "#{date}", time.Now().Format(time.DateTime))
	// 公共: 输出目录
	output, ok := environment.Get("P_OUTPUT")
	if ok {
		content = strings.ReplaceAll(content, "#{output}", output)
	}
	// 公共: 执行前
	before, ok := environment.Get("P_PACKAGE_BEFORE")
	if ok && before != "" {
		content = strings.ReplaceAll(content, "#{before}", "&& "+before)
	}
	// 公共: 执行后
	after, ok := environment.Get("P_PACKAGE_AFTER")
	if ok && after != "" {
		content = strings.ReplaceAll(content, "#{after}", "&& "+after)
	}

	// 前端: 构建命令
	buildCommand, ok := environment.Get("P_BUILD_CMD")
	if ok {
		content = strings.ReplaceAll(content, "#{buildCommand}", buildCommand)
	} else {
		content = strings.ReplaceAll(content, "#{buildCommand}", "cnpm")
	}

	// 特殊符号
	content = strings.ReplaceAll(content, "@#", "$")

	// 环境变量替换
	for key, value := range environment.LocalMap() {
		content = strings.ReplaceAll(content, "#{"+key+"}", value)
	}

	// 如果还有变量则替换空
	for {
		rule := regexp.MustCompile(`#{[a-zA-Z0-9_-]*}`)
		ruleResult := rule.FindStringSubmatch(content)
		if ruleResult == nil {
			break
		}
		for _, param := range ruleResult {
			content = strings.ReplaceAll(content, param, "")
		}
	}
	return content
}
