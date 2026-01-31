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
- build-installer workflow，自动生成 install.yaml 并创建 PR
- NuwaStatus 增强字段
  - `observedGeneration`：已处理的 spec 版本
  - `updatedReplicas`：已更新的副本数
  - `serviceIP`：Service ClusterIP
  - `loadBalancerIP`：LoadBalancer 外部 IP
  - `pvcStatus`：PVC 状态（Bound/Pending/NotFound）
- Conditions 支持
  - `CloneSetReady`：CloneSet 是否就绪
  - `ServiceReady`：Service 是否创建成功
  - `PVCReady`：PVC 是否绑定成功
  - `Progressing`：是否正在滚动更新
- Dependabot 自动合并 workflow（botmerge）
- Service 配置增强（`service` 字段）
  - `type`：Service 类型（ClusterIP/NodePort/LoadBalancer）
  - `annotations`：Service 注解
  - `loadBalancerClass`：LoadBalancer 实现类
  - `loadBalancerIP`：指定 LoadBalancer IP
  - `externalTrafficPolicy`：外部流量策略

### Changed
- CI 添加 `latest` tag 支持，push 到 master 时自动生成
- README 添加 YAML 快速安装方式（`kubectl apply -f dist/install.yaml`）
- README 添加状态字段和 Conditions 说明
- CI 执行顺序优化：lint/test → build → docker → e2e
- E2E 测试使用 `helm/kind-action` 简化 Kind 集群创建
- build-installer workflow 简化 PR 创建逻辑
- CI 构建优化：先编译多架构二进制，Dockerfile 只拷贝（加速镜像构建）
- E2E 测试流程优化：安装 OpenKruise 并配置 containerd socket
- Service 配置重构：`serviceType` 改为嵌套的 `service` 字段

### Fixed
- 修复 `containerPorts` 和 `servicePorts` 切片 prealloc lint 警告
- 修复 `.dockerignore` 排除预编译二进制的问题
