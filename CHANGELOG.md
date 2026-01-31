# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2026-01-31]

### Added
- 初始化 kubebuilder 项目，API Group: `app.12306.work/v1`
- Nuwa CRD 定义，支持简化的应用部署配置
  - 镜像配置（image）
  - 副本数（replicas）
  - 环境变量（env）
  - 端口映射（ports），自动创建 Service
  - 资源限制（resources）
  - 命令和参数（command/args）
  - 镜像拉取策略和凭证
- Controller 实现，自动创建 OpenKruise CloneSet 和 Service
- 多种存储类型支持
  - PVC：持久卷声明
  - EmptyDir：临时存储
  - HostPath：主机路径挂载
- GitHub Actions CI 配置
  - Lint 代码检查
  - 单元测试
  - 构建验证
  - Docker 多架构镜像构建（amd64/arm64）
  - E2E 测试（Kind 集群）
- Taskfile 快捷命令封装
- 中文 README 文档
