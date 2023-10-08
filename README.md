# Golang Bpp Client (女娲)
> `女娲` 项目名称来源: 女娲造人

> Golang 实现的`构建脚本`需要结合（Bpp Server）一起使用

## 功能描述
> 客户端，编译输出`bpp` 程序，提供脚本结合`GitLab CI/CD` 完成构建
> - 使用容器隔离构建每个应用
> - 支持`Ali Tencent Nacos` 配置的同步
> - 支持服务构建脚本配置（超多环境变量）
> - 支持 Docker 镜像管理，推送完成后删除，不占用空间
> - 支持 API 动态更新Kubernetes 集群服务
> - 支持指定跳过构建步骤
> - 支持企业微信消息（成功、失败）消息推送

## 使用方法
```bash
# 查看帮助程序
bpp --help
```


## 鸣谢
> 作者：C猫

> 邮箱：735802488@qq.com