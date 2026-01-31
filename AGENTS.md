# Nuwa - AI Agent Guide

## 项目概述

Nuwa 是一个 Kubernetes CRD 控制器，基于 OpenKruise CloneSet 实现简化的应用部署。

**核心功能**：只需配置镜像、环境变量、端口、存储等基本信息，自动创建 CloneSet 和 Service。

## 项目结构

```
api/v1/
├── nuwa_types.go           # CRD 类型定义（核心文件）
├── groupversion_info.go    # API Group 信息
└── zz_generated.deepcopy.go # 自动生成（勿编辑）

internal/controller/
└── nuwa_controller.go      # Controller 实现（核心文件）

config/
├── crd/bases/              # 生成的 CRD YAML（勿编辑）
├── rbac/                   # 生成的 RBAC（勿编辑）
├── samples/                # 示例 CR（可编辑）
└── manager/                # 部署配置

cmd/main.go                 # 入口文件
Makefile                    # 构建命令（完整）
Taskfile.yml                # 快捷命令（精简）
```

## CRD 字段说明

```go
// NuwaSpec 核心字段
type NuwaSpec struct {
    Image           string                    // 必填：容器镜像
    Replicas        *int32                    // 副本数，默认 1
    Env             []corev1.EnvVar           // 环境变量
    Ports           []PortMapping             // 端口映射，自动创建 Service
    Storage         *StorageSpec              // 存储配置
    Resources       *ResourceRequirements     // 资源限制
    ServiceType     corev1.ServiceType        // Service 类型
    ImagePullPolicy corev1.PullPolicy         // 镜像拉取策略
    ImagePullSecrets []LocalObjectReference   // 私有镜像凭证
    Command         []string                  // 覆盖 entrypoint
    Args            []string                  // 容器参数
}

// StorageSpec 存储类型
type StorageSpec struct {
    Type             StorageType  // PVC（默认）、EmptyDir、HostPath
    MountPath        string       // 挂载路径（必填）
    Size             string       // 存储大小
    StorageClassName *string      // 存储类（仅 PVC）
    AccessModes      []string     // 访问模式（仅 PVC）
    HostPath         string       // 主机路径（仅 HostPath）
    HostPathType     *string      // HostPath 类型
}
```

## 开发指南

### 修改 CRD 后

```bash
make manifests  # 重新生成 CRD/RBAC
make generate   # 重新生成 DeepCopy
make build      # 编译验证
```

### 常用命令

```bash
# 开发
make run        # 本地运行
make test       # 单元测试
make lint       # 代码检查

# 部署
make install    # 安装 CRD
make deploy IMG=<image>  # 部署控制器
make undeploy   # 卸载控制器
```

### 快捷命令（Taskfile）

```bash
task build      # 编译
task run        # 本地运行
task test       # 测试
task deploy     # 部署
```

## Controller 逻辑

### Reconcile 流程

1. **获取 Nuwa 资源** → 不存在则返回
2. **处理删除** → 移除 Finalizer
3. **添加 Finalizer** → 确保清理
4. **Reconcile CloneSet** → 创建/更新 OpenKruise CloneSet
5. **Reconcile Service** → 如有端口配置，创建/更新 Service
6. **更新 Status** → 同步 CloneSet 状态

### 关键实现

```go
// 存储类型处理
switch storageType {
case StorageTypePVC:
    // 使用 VolumeClaimTemplates
case StorageTypeEmptyDir:
    // 使用 Volumes + EmptyDir
case StorageTypeHostPath:
    // 使用 Volumes + HostPath
}
```

## 依赖

- **OpenKruise**：底层使用 CloneSet 管理 Pod
- **controller-runtime**：Kubernetes controller 框架

## 注意事项

### 勿编辑的文件
- `config/crd/bases/*.yaml` - 由 `make manifests` 生成
- `config/rbac/role.yaml` - 由 `make manifests` 生成
- `**/zz_generated.*.go` - 由 `make generate` 生成

### 保留 Scaffold 标记
不要删除 `// +kubebuilder:scaffold:*` 注释，CLI 依赖这些标记注入代码。

## 参考资料

- [Kubebuilder Book](https://book.kubebuilder.io)
- [OpenKruise CloneSet](https://openkruise.io/docs/user-manuals/cloneset)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
