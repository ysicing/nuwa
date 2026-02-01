# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2026-02-01]

### Added
- 资源预设配置功能（ResourcesPreset）
  - 支持 8 种预设规格：none/nano/micro/small/medium/large/xlarge/xxlarge
  - 自动计算 CPU 和内存资源限制
- 资源超卖配置（ResourcesOvercommit）
  - 支持 3 种超卖级别：none/low/high
  - 自动计算 requests 与 limits 的比例
- 更新策略配置（UpdateStrategy）
  - 支持 3 种更新类型：InPlaceIfPossible/ReCreate/InPlaceOnly
  - 支持灰度发布参数：partition/maxUnavailable/maxSurge/gracePeriodSeconds
- **PVC 保留策略配置（PVCRetainPolicy）**
  - 支持 `Retain`（保留，默认）和 `Delete`（删除）两种策略
  - 通过 `storage.retainPolicy` 字段配置
  - `Retain` 策略：删除 Nuwa 时 PVC 保留，确保数据安全
  - `Delete` 策略：删除 Nuwa 时 PVC 自动删除，适合临时环境
  - **支持安全方向的动态策略变更**：Delete → Retain 立即生效
  - Retain → Delete 需要删除重建（防止意外数据丢失）
- 完整的单元测试覆盖
  - `buildUpdateStrategy` 函数测试（9 个测试用例）
  - `getResourcesFromPreset` 函数测试（18 个测试用例）
  - 覆盖所有 preset 类型和 overcommit 级别
- 详细的配置文档
  - `docs/resources-configuration.md`：资源配置指南
  - `docs/storage-configuration.md`：存储配置指南
  - `docs/pvc-retention.md`：PVC 保留策略文档

### Changed
- **重要**：PVC 存储实现方式变更
  - 从 CloneSet volumeClaimTemplates 改为独立创建 PVC
  - 所有副本共享同一个 PVC（`<nuwa-name>-pvc`）
  - **可配置的 PVC 保留策略**：
    - `Retain`（默认）：删除 Nuwa 时 PVC 保留，确保数据安全
    - `Delete`：删除 Nuwa 时 PVC 自动删除，适合临时环境
  - PVC 添加 `nuwa.12306.work/created-by` 和 `nuwa.12306.work/retain-policy` 注解
- 资源配置默认值策略
  - `resourcesPreset` 默认为 `none`（向后兼容，不影响现有 CR）
  - `resourcesOvercommit` 默认为 `none`（无超卖）
  - 用户需显式设置才会应用资源限制
- CI workflow 优化
  - 合并 Build Installer 工作流到主 CI 流程
  - E2E 测试使用 artifact 机制，避免重复构建
  - 添加 `paths-ignore` 忽略 `dist/install.yaml` 更新
  - 自动创建 PR 更新 install.yaml
- 代码质量改进
  - 提取 `buildLabels()` 辅助函数，消除重复代码
  - 提取 `getPVCName()` 辅助函数，统一 PVC 命名
  - 将 `presetLimits` 提取为包级别常量，提升性能
  - 优化 `updateStatus` 函数，重新获取最新资源避免冲突

### Fixed
- **关键**：PVC 创建顺序错误
  - PVC 现在在 CloneSet 之前创建，避免 Pod Pending
- **关键**：存储大小格式验证
  - 添加 `resource.ParseQuantity` 验证，避免 `MustParse` panic
  - `buildPVC` 函数返回 error，提升控制器健壮性
- **关键**：控制器状态更新冲突
  - `updateStatus` 函数重新获取最新 Nuwa 资源
  - 避免 "object has been modified" 错误
  - 使用最新 ResourceVersion 进行状态更新
- 资源计算精度问题
  - 使用 `SetMilli(MilliValue() / divisor)` 替代 `Set(Value() / divisor)`
  - 修复亚核心值（100m, 250m, 500m）的 CPU requests 计算错误
- EmptyDir 默认大小问题
  - 移除 CRD 中 Storage.Size 的默认值
  - 只对 PVC 类型在控制器中应用默认 8Gi
- 测试单位错误
  - 修正内存测试中 `MilliValue()` 为 `Value()`
  - 正确比较字节数而非毫字节数
- Sample YAML 字段更新
  - 修复 `serviceType` 为嵌套的 `service.type` 结构
  - 确保 E2E 测试使用正确的 API 字段

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
