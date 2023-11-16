# =================================================================
# Message 消息推送脚本
# 消息  => 企业微信
# =================================================================

set -e

# 服务器读取脚本
SERVER=127.0.0.1:8080

# =======================================
# 内容转义 去掉非法字符
function escape() {
  if [ -z "$1" ]
  then
    echo ""
  else
    echo "$1" | sed -e 's/\*//g' -e 's/> //g' -e 's/\n/ /g' -e 's/<br\/>/\n/g' -e 's/%/ /g' | tr '\n' ' '
  fi
}


# =======================================
# 获取Key服务器的内容
function env() {
  if [ -z "$1" ]
  then
    echo ""
  else
    data=$(curl $SERVER/pair/"$(echo "$1" | tr '[:lower:]' '[:upper:]')" | jq .data)
    echo "${data:1:${#data}-2}"
  fi
}


# 消息状态
STATUS=$1
# 消息内容
MESSAGE=$2


if [ -z "$1" ]; then
    echo "Error: Argument not provided. Exiting."
    exit 1
fi

if [ -z "$2" ]; then
    echo "Error: Argument not provided. Exiting."
    exit 1
fi

# 加载Args 动态参数
for arg in "${@:3}"; do
  echo "$arg"
  declare "$arg"
done

# 机器人地址
ROBOT=$(env "GL_MESSAGE_CP_WECHAT_ROBOT")

# 如果是构建成功则输出成功消息
# shellcheck disable=SC2034
if [ "$STATUS" = "true" ]; then
  title='<font color="info">构建成功</font>'
else
  title='<font color="warning">构建失败</font>'
fi

# 如果是构建失败输出原因
# shellcheck disable=SC2034
if [ "$STATUS" = "true" ]; then
  failReason=''
else
  failReason="\r\n> **失败原因**：[点击查看失败详情](${CI_JOB_URL}) ${MESSAGE}"
fi

# shellcheck disable=SC2034
projectName="$P_PROJECT_NAME"
# shellcheck disable=SC2034
author="$CI_COMMIT_AUTHOR"

env_value=$(echo "$P_COLONY_ENV" | tr '[:lower:]' '[:upper:]')
# 如果 P_COLONY_ENV 变量不存在，则尝试使用 colonyEnv
if [ -z "$env_value" ]; then
    env_value=$(echo "$colonyEnv" | tr '[:lower:]' '[:upper:]')
fi
case $(echo "$env_value" | tr '[:lower:]' '[:upper:]') in
    "DEV")
        env="开发环境"
        ;;
    "TEST")
        env="测试环境"
        ;;
    "PREV")
        env="预发布环境"
        ;;
    "PROD")
        env="生产环境"
        ;;
    *)
        # 如果 $env_value 为空，则设置 env 为 "未知环境"，否则设置为 $env_value
        if [ -z "$env_value" ]; then
            env="未知环境"
        else
            env="$env_value"
        fi
        ;;
esac

# shellcheck disable=SC2034
commitRefName="$CI_COMMIT_REF_NAME"
# shellcheck disable=SC2034
description="$CI_COMMIT_MESSAGE"
# shellcheck disable=SC2034
commitId="$CI_COMMIT_SHA"
# shellcheck disable=SC2034
link="$CI_JOB_URL"


# 所有变量
CONTENT=$(env "GL_MESSAGE_CP_WECHAT_TEMPLATE")

projectName=$(escape "${projectName}")
author=$(escape "${author}")
commitRefName=$(escape "${commitRefName}")
description=$(escape "${description}")

CONTENT=$(echo -n "$CONTENT" | sed "s%#{title}%${title}%g")
CONTENT=$(echo -n "$CONTENT" | sed "s%#{failReason}%${failReason}%g")
echo -e "projectName: ${projectName}"
CONTENT=$(echo -n "$CONTENT" | sed "s%#{projectName}%${projectName}%g")
echo -e "author: ${projectName}"
CONTENT=$(echo -n "$CONTENT" | sed "s%#{author}%${author}%g")
CONTENT=$(echo -n "$CONTENT" | sed "s%#{env}%${env}%g")
echo -e "commitRefName: ${commitRefName}"
CONTENT=$(echo -n "$CONTENT" | sed "s%#{commitRefName}%${commitRefName}%g")
echo -e "description ${description}"
CONTENT=$(echo -n "$CONTENT" | sed "s%#{description}%${description}%g")
CONTENT=$(echo -n "$CONTENT" | sed "s%#{commitId}%${commitId}%g")
CONTENT=$(echo -n "$CONTENT" | sed "s%#{link}%${link}%g")

# 构建参数
REQUEST='{"msgtype": "markdown", "markdown": { "content": "#{content}" }}'
CONTENT=${CONTENT//"\""/"\\\""}
REQUEST=${REQUEST//"#{content}"/"$CONTENT"}

# 打印发送消息内容
echo -e "$CONTENT"

# 发送消息
curl -X POST -H "Content-Type: application/json" -d "$REQUEST" "$ROBOT"
