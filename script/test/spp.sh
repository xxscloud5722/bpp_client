set -e

# 服务器读取脚本
SERVER=127.0.0.1:8080

function error_handler() {
  echo "Error occurred. Custom action performed."
}

trap 'error_handler' ERR

function env() {
  if [ -z "$1" ]
  then
    echo ""
  else
    curl $SERVER/pair/"$(echo "$1" | tr '[:lower:]' '[:upper:]')" | jq -r .data
  fi
}

function check() {
    if [ -z "$1" ]
    then
      echo "参数(${2}) 没有预设值"
      exit 1
    fi
}

function parse() {
  CONTEXT=$1
  # 前端
  if [ -n "$localPath" ]; then
      CONTEXT=${CONTEXT//"#{localWebPath}"/"$localPath"}

      # 路径分割链接
      IFS='/' read -ra pathArray <<< "$localPath"
      pathArray=("${pathArray[@]:0:${#pathArray[@]}-1}")
      CONTEXT=${CONTEXT//"#{localRootPath}"/"/app/$(IFS="/" ; echo "${pathArray[*]}")"}

      CONTEXT=${CONTEXT//"#{webPath}"/"/$localPath"}
  else
      CONTEXT=${CONTEXT//"#{localWebPath}"/}
      CONTEXT=${CONTEXT//"#{localRootPath}"/}
      CONTEXT=${CONTEXT//"#{webPath}"/}
  fi

  # 公共: 替换工作目录
  CONTEXT=${CONTEXT//"#{working}"/"${CI_PROJECT_DIR}"}

  # 公共: 项目ID
  CONTEXT=${CONTEXT//"#{id}"/"${CI_PROJECT_ID}"}

  # 公共: 项目名称
  CONTEXT=${CONTEXT//"#{name}"/"${P_SERVICE_NAME}"}

  # 公共: 提交ID
  CONTEXT=${CONTEXT//"#{commitId}"/"${CI_COMMIT_SHA}"}

  # 公共: 构建时间
  CONTEXT=${CONTEXT//"#{date}"/"$(date +"%Y-%m-%d %H:%M:%S")"}

  # 公共: 输出目录
  if [ -n "$P_OUTPUT" ]
  then
    CONTEXT=${CONTEXT//"#{output}"/"$P_OUTPUT"}
  fi

  # 公共: 执行前
  if [ -n "$P_PACKAGE_BEFORE" ]
  then
    CONTEXT=${CONTEXT//"#{before}"/"&& $P_PACKAGE_BEFORE"}
  fi

  # 公共: 执行后
  if [ -n "$P_PACKAGE_AFTER" ]
  then
    CONTEXT=${CONTEXT//"#{after}"/"&& $P_PACKAGE_AFTER"}
  fi

  # 前端: 构建命令
  if [ -n "$P_BUILD_CMD" ]
  then
    CONTEXT=${CONTEXT//"#{buildCommand}"/"$P_BUILD_CMD"}
  else
    CONTEXT=${CONTEXT//"#{buildCommand}"/"cnpm"}
  fi

  # 特殊符号
  CONTEXT=${CONTEXT//"@#"/"$"}

  # 所有变量
  env_variables=$(env)
  for var in $env_variables; do
    key=$(echo "$var" | cut -d'=' -f1)
    value=$(echo "$var" | cut -d'=' -f2-)

    CONTEXT=${CONTEXT//"#{${key}}"/"${value}"}
  done

  # 替换参数为空字符串
  # shellcheck disable=SC2001
  CONTEXT=$(echo "$CONTEXT" | sed 's/#{[a-zA-Z0-9_-]*}//g')
  echo "$CONTEXT"
}

function docker_auth() {
  file_path="/root/$P_DOCKER_AUTH_TYPE"
  if [ -e "$file_path" ]; then
    echo "$file_path"
  else
      config=$(env "GL_DOCKER_AUTH_$P_DOCKER_AUTH_TYPE")
      if [ -n "$config" ]
      then
        IFS=','
        read -ra values <<< "$config"
        value_count=${#values[@]}
        if [ "$value_count" -eq 2 ]; then
          hostname="registry.cn-shanghai.aliyuncs.com"
          username="${values[0]}"
          password="${values[1]}"
        elif [ "$value_count" -eq 3 ]; then
          username="${values[0]}"
          password="${values[1]}"
          hostname="${values[2]}"
        else
          echo "Docker Auth is not supported"
          exit 1
        fi
      fi
      docker --config "$file_path" login "$hostname" -u "$username" -p "$password"
      echo "$file_path"
    fi
}

function package() {

  check "$P_PACKAGE_TYPE" 'P_PACKAGE_TYPE'
  check "$P_SERVICE_NAME" 'P_SERVICE_NAME'

  # 基础脚本
  S_SCRIPT=" set -e \n function error_handler() { \n echo 'Error occurred. Custom action performed.' \n }"
  S_SCRIPT="$S_SCRIPT \n trap 'error_handler' ERR"
  S_SCRIPT="$S_SCRIPT \n mkdir -p /opt/repository/$CI_PROJECT_ID"

  # 外部脚本
  PACKAGE_SCRIPT=$(env "GL_BUILD_SCRIPT_$P_PACKAGE_TYPE")
  if [ -n "$PACKAGE_SCRIPT" ]
  then
    PACKAGE_SCRIPT=$(parse "$PACKAGE_SCRIPT")
    S_SCRIPT="$S_SCRIPT \n $PACKAGE_SCRIPT"
  fi

  # 构建脚本
  echo "ImageName: $P_IMAGE_NAME"
  if [ -n "$P_IMAGE_NAME" ]
  then
    S_SCRIPT="$S_SCRIPT \n docker build -t $P_IMAGE_NAME ."
  fi

  # 写出到脚本文件
  echo -e "$S_SCRIPT" > 'build.sh' && chmod +x build.sh

  # 输出日志
  echo "================ Deploy Script ================="
  echo -e "$S_SCRIPT"
  echo "================================================"


  # 配置文件输出
  PACKAGE_CONFIG=$(env "GL_BUILD_CONFIG_$P_PACKAGE_TYPE")
  if [ -n "$PACKAGE_CONFIG" ]
  then
    PACKAGE_CONFIG=$(parse "$PACKAGE_CONFIG")
    PACKAGE_CONFIG_FILE_NAME=${PACKAGE_CONFIG%%#*}
    PACKAGE_CONFIG_FILE=${PACKAGE_CONFIG#*#}
    # 写出配置文件
    echo -e "$PACKAGE_CONFIG_FILE" > "$PACKAGE_CONFIG_FILE_NAME"
    # 输出日志
    echo "================ $PACKAGE_CONFIG_FILE_NAME ================="
    echo -e "$PACKAGE_CONFIG_FILE"
    echo "================================================"
  fi

  # 配置公共文件输出
  P_PACKAGE_TYPE=$(echo "$P_PACKAGE_TYPE" | tr '[:lower:]' '[:upper:]')
  if [[ $P_PACKAGE_TYPE =~ ^VUE || $P_PACKAGE_TYPE =~ ^REACT || $P_PACKAGE_TYPE =~ ^WEB_ ]]
  then
    P_PACKAGE_TYPE_COMMON="NGINX"
  elif [[ $P_PACKAGE_TYPE =~ ^MVN ]]; then
    P_PACKAGE_TYPE_COMMON="JAVA"
  else
    P_PACKAGE_TYPE_COMMON=""
  fi
  if [ -n "$P_PACKAGE_TYPE_COMMON" ]
  then
    PACKAGE_COMMON_CONFIG=$(env "GL_BUILD_CONFIG_$P_PACKAGE_TYPE_COMMON")
    if [ -n "$PACKAGE_COMMON_CONFIG" ]
    then
      PACKAGE_COMMON_CONFIG=$(parse "$PACKAGE_COMMON_CONFIG")
      PACKAGE_CONFIG_FILE_NAME=${PACKAGE_COMMON_CONFIG%%#*}
      PACKAGE_CONFIG_FILE=${PACKAGE_COMMON_CONFIG#*#}
      # 写出配置文件
      echo -e "$PACKAGE_CONFIG_FILE" > "$PACKAGE_CONFIG_FILE_NAME"
      # 输出日志
      echo "================ $PACKAGE_CONFIG_FILE_NAME ================="
      echo -e "$PACKAGE_CONFIG_FILE"
      echo "================================================"
    fi
  fi

  # Dockerfile
  DOCKER_FILE=$(env "GL_BUILD_DOCKERFILE_$P_PACKAGE_TYPE")
  if [ -n "$DOCKER_FILE" ]
  then
    DOCKER_FILE=$(parse "$DOCKER_FILE")
    # 写出配置文件
    echo "$DOCKER_FILE" > 'Dockerfile'
    # 输出日志
    echo "================ Dockerfile ================="
    echo "$DOCKER_FILE"
    echo "================================================"
  fi
}

function push() {
  check "$P_IMAGE_NAME" 'P_IMAGE_NAME'
  config_path=$(docker_auth)
  echo "Image: $P_IMAGE_NAME"
  docker --config "$config_path" push "$P_IMAGE_NAME"
}

function pull() {
  check "$P_IMAGE_NAME" 'P_IMAGE_NAME'
  config_path=$(docker_auth)
  echo "Image: $P_IMAGE_NAME"
  docker --config "$config_path" pull "$P_IMAGE_NAME"
}

function tag() {
  # shellcheck disable=SC2154
  check "$oldImageName" 'oldImageName'
  # shellcheck disable=SC2154
  check "$newImageName" 'newImageName'
  docker tag "$oldImageName" "$newImageName"
}

function remove() {
  check "$P_IMAGE_NAME" 'P_IMAGE_NAME'
  docker rmi -f "$P_IMAGE_NAME"
}

function release() {
  echo "release ..."
}

function nacos_sync() {
  echo "nacos_sync ..."
}

function ssh() {
  echo "ssh"
  check "$P_OUTPUT" 'P_OUTPUT'
  # shellcheck disable=SC2154
  check "$server" 'server'
  # shellcheck disable=SC2154
  check "$path" 'path'

  # 获取用户
  server=$(env "GS_SERVER_$server")
  rows=${server#*#}
  IFS=$'\n' read -ra array <<< "$rows"
  host="${array[0]}"
  port="${array[1]}"
  username="${array[2]}"
  password="${array[3]}"

  # 压缩文件
  tar -czvf install.tar.gz "$P_OUTPUT"

  uuid=$(uuidgen)

  # 上传前
  ssh_a=" mkdir -p ${path} \n"
  ssh_a="$ssh_a mv -f ${path} /tmp/gitlab_${uuid} \n"
  ssh_a="$ssh_a mkdir -p ${path} \n"
  # shellcheck disable=SC2154
  ssh_a="$ssh_a $before"

  echo "==================== SSH-A ====================="
  echo -e "$ssh_a"
  echo "================================================"
  echo -e "$ssh_a" > 'ssh_a.sh'
  sshpass -p "$password" ssh "${username}@${host}" 'bash -s' < ssh_a.sh

  # 上传文件
  sshpass -p "$password" scp -P "${port}" ./install.tar.gz "${username}@${host}:${path}/install.tar.gz"

  # 上传后
  ssh_b=" cd ${path} \n"
  ssh_b="$ssh_b tar -zxvf ./install.tar.gz \n"
  ssh_b="$ssh_b mv -f ./install.tar.gz /tmp/gitlab_${uuid} \n"
  # shellcheck disable=SC2154
  ssh_b="$ssh_b $after"

  echo "==================== SSH-B ====================="
  echo -e "$ssh_b"
  echo "================================================"
  echo -e "$ssh_b" > 'ssh_b.sh'
  sshpass -p "$password" ssh "${username}@${host}" 'bash -s' < ssh_b.sh
}

# shellcheck disable=SC2181
if [ $? -ne 0 ]; then
  echo "Error: Last command failed. Exiting."
  exit 1
fi

if [ -z "$1" ]; then
    echo "Error: Argument not provided. Exiting."
    exit 1
fi

# 加载Args 动态参数
for arg in "${@:2}"; do
  echo "$arg"
  eval "$arg"
done

# 切换目录
cd "$CI_PROJECT_DIR"
# 输出路径
echo "Local Path: $(pwd)"

if [ "$1" == "package" ];
then
  package
elif [ "$1" == "push" ];
then
  push
elif [ "$1" == "pull" ];
then
  pull
elif [ "$1" == "tag" ];
then
  tag
elif [ "$1" == "remove" ];
then
  remove
elif [ "$1" == "release" ];
then
  release
elif [ "$1" == "nacosSync" ];
then
  nacos_sync
fi
