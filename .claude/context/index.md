# 📂 Nuwa 功能-代码映射报告

## 🏗️ 项目概览
- **技术栈**: Go + Kubebuilder + controller-runtime
- **架构模式**: Kubernetes Operator Pattern (CRD + Controller)
- **底层依赖**: OpenKruise CloneSet
- **API Group**: `app.12306.work/v1`
- **构建工具**: Make + Go Build
- **包管理**: Go Modules

## 📊 功能模块统计
- **CRD 类型定义**: 4 个 (Nuwa, NuwaSpec, PortMapping, StorageSpec)
- **Controller**: 1 个 (NuwaReconciler)
- **配置清单**: 15+ 个 (CRD, RBAC, Deployment)
- **测试文件**: 4 个 (单元测试 + E2E)

## 🗂️ 目录结构概览
```
nuwa/
├── api/v1/                    # CRD 类型定义
│   ├── nuwa_types.go          # 核心类型
│   └── groupversion_info.go   # API Group 信息
├── internal/controller/       # Controller 实现
│   └── nuwa_controller.go     # 核心逻辑
├── cmd/main.go                # 程序入口
├── config/                    # K8s 配置
│   ├── crd/bases/             # CRD YAML
│   ├── rbac/                  # RBAC 配置
│   ├── samples/               # 示例 CR
│   └── manager/               # 部署配置
└── Makefile                   # 构建命令
```

---

## 🎯 功能映射表

### CRD 配置 - 镜像设置

**🔤 用户描述方式**:
- 主要: "设置镜像", "修改容器镜像", "更换镜像"
- 别名: "image", "容器镜像", "镜像地址", "镜像版本"

**📍 代码位置**:
- 类型定义: `api/v1/nuwa_types.go:28` - `Image string` 字段
- Controller 使用: `internal/controller/nuwa_controller.go:175` - 构建 Container

**🎨 CR 示例**:
```yaml
spec:
  image: nginx:latest
```

**⚡ 修改指引**:
- 添加镜像验证: 编辑 `api/v1/nuwa_types.go` 添加 `+kubebuilder:validation` 标记
- 修改默认值: 添加 `+kubebuilder:default` 标记
- 修改使用逻辑: 编辑 `nuwa_controller.go` 的 `buildCloneSet` 函数

---

### CRD 配置 - 副本数

**🔤 用户描述方式**:
- 主要: "设置副本数", "修改实例数量", "扩缩容"
- 别名: "replicas", "副本", "实例数", "Pod 数量"

**📍 代码位置**:
- 类型定义: `api/v1/nuwa_types.go:34` - `Replicas *int32` 字段
- Controller 使用: `internal/controller/nuwa_controller.go:144-147` - 设置 CloneSet 副本

**🎨 CR 示例**:
```yaml
spec:
  replicas: 3
```

**⚡ 修改指引**:
- 修改默认值: 编辑 `nuwa_types.go` 的 `+kubebuilder:default=1`
- 添加最大限制: 添加 `+kubebuilder:validation:Maximum=100`
- 修改扩缩容逻辑: 编辑 `buildCloneSet` 函数

---

### CRD 配置 - 环境变量

**🔤 用户描述方式**:
- 主要: "设置环境变量", "添加 env", "配置环境"
- 别名: "env", "环境变量", "environment", "配置项"

**📍 代码位置**:
- 类型定义: `api/v1/nuwa_types.go:38` - `Env []corev1.EnvVar` 字段
- Controller 使用: `internal/controller/nuwa_controller.go:178` - 传递给 Container

**🎨 CR 示例**:
```yaml
spec:
  env:
    - name: TZ
      value: Asia/Shanghai
    - name: DB_HOST
      valueFrom:
        secretKeyRef:
          name: db-secret
          key: host
```

**⚡ 修改指引**:
- 支持 envFrom: 在 `nuwa_types.go` 添加 `EnvFrom` 字段
- 修改传递逻辑: 编辑 `buildCloneSet` 中的 Container 构建

---

### CRD 配置 - 端口映射

**🔤 用户描述方式**:
- 主要: "配置端口", "设置端口映射", "暴露端口"
- 别名: "ports", "端口", "port mapping", "服务端口"

**📍 代码位置**:
- 类型定义: `api/v1/nuwa_types.go:42,76-100` - `Ports []PortMapping` 和 `PortMapping` 结构
- Service 创建: `internal/controller/nuwa_controller.go:271-321` - `buildService` 函数
- Container 端口: `internal/controller/nuwa_controller.go:154-170` - 构建 containerPorts

**🎨 CR 示例**:
```yaml
spec:
  ports:
    - name: http
      containerPort: 80
      servicePort: 80
    - name: metrics
      containerPort: 9090
      nodePort: 30090
  serviceType: NodePort
```

**⚡ 修改指引**:
- 添加新端口字段: 编辑 `PortMapping` 结构体
- 修改 Service 创建逻辑: 编辑 `buildService` 函数
- 修改默认协议: 编辑 `+kubebuilder:default=TCP`

---

### CRD 配置 - 存储配置

**🔤 用户描述方式**:
- 主要: "配置存储", "设置持久化", "挂载卷"
- 别名: "storage", "存储", "PVC", "volume", "持久卷"

**📍 代码位置**:
- 类型定义: `api/v1/nuwa_types.go:46,102-141` - `Storage *StorageSpec` 和 `StorageSpec` 结构
- 存储类型: `api/v1/nuwa_types.go:102-109` - `StorageType` 枚举 (PVC/EmptyDir/HostPath)
- Controller 处理: `internal/controller/nuwa_controller.go:187-252` - 存储类型分支处理

**🎨 CR 示例**:
```yaml
# PVC 持久存储
spec:
  storage:
    type: PVC
    size: 10Gi
    mountPath: /data
    storageClassName: standard

# EmptyDir 临时存储
spec:
  storage:
    type: EmptyDir
    mountPath: /tmp/cache
    size: 1Gi

# HostPath 主机路径
spec:
  storage:
    type: HostPath
    mountPath: /data
    hostPath: /mnt/data
```

**⚡ 修改指引**:
- 添加新存储类型: 在 `StorageType` 枚举添加新值，在 `buildCloneSet` 添加 case 分支
- 修改 PVC 模板: 编辑 `case appv1.StorageTypePVC` 分支
- 添加多卷支持: 将 `Storage *StorageSpec` 改为 `Storages []StorageSpec`

---

### CRD 配置 - 资源限制

**🔤 用户描述方式**:
- 主要: "设置资源限制", "配置 CPU 内存", "资源配额"
- 别名: "resources", "资源", "limits", "requests", "CPU", "内存"

**📍 代码位置**:
- 类型定义: `api/v1/nuwa_types.go:50` - `Resources *corev1.ResourceRequirements`
- Controller 使用: `internal/controller/nuwa_controller.go:183-185` - 设置 Container Resources

**🎨 CR 示例**:
```yaml
spec:
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
```

**⚡ 修改指引**:
- 添加默认资源: 在 Controller 中添加默认值逻辑
- 添加资源验证: 添加 webhook 验证资源配置合理性

---

### CRD 配置 - Service 类型

**🔤 用户描述方式**:
- 主要: "设置 Service 类型", "修改服务类型", "暴露方式"
- 别名: "serviceType", "ClusterIP", "NodePort", "LoadBalancer"

**📍 代码位置**:
- 类型定义: `api/v1/nuwa_types.go:73` - `ServiceType corev1.ServiceType`
- Controller 使用: `internal/controller/nuwa_controller.go:304-307` - 设置 Service Type

**🎨 CR 示例**:
```yaml
spec:
  serviceType: LoadBalancer  # 或 ClusterIP, NodePort
```

**⚡ 修改指引**:
- 修改默认类型: 编辑 `+kubebuilder:default=ClusterIP`
- 添加新类型支持: 编辑 `+kubebuilder:validation:Enum`

---

### Controller 行为 - CloneSet 创建

**🔤 用户描述方式**:
- 主要: "修改 CloneSet 创建逻辑", "调整 Pod 模板"
- 别名: "CloneSet", "Pod 模板", "工作负载"

**📍 代码位置**:
- 主函数: `internal/controller/nuwa_controller.go:120-141` - `reconcileCloneSet`
- 构建函数: `internal/controller/nuwa_controller.go:143-261` - `buildCloneSet`

**⚡ 修改指引**:
- 添加 Pod 注解: 在 `buildCloneSet` 的 `Template.ObjectMeta` 添加 Annotations
- 添加亲和性: 在 `PodSpec` 添加 `Affinity` 字段
- 添加探针: 在 Container 添加 `LivenessProbe`/`ReadinessProbe`

---

### Controller 行为 - Service 创建

**🔤 用户描述方式**:
- 主要: "修改 Service 创建逻辑", "调整服务配置"
- 别名: "Service", "服务", "网络"

**📍 代码位置**:
- 主函数: `internal/controller/nuwa_controller.go:263-285` - `reconcileService`
- 构建函数: `internal/controller/nuwa_controller.go:287-337` - `buildService`

**⚡ 修改指引**:
- 添加 Service 注解: 在 `buildService` 的 `ObjectMeta` 添加 Annotations
- 添加 Session Affinity: 在 `ServiceSpec` 添加 `SessionAffinity`

---

### Controller 行为 - 状态更新

**🔤 用户描述方式**:
- 主要: "修改状态更新", "调整 Status", "状态同步"
- 别名: "status", "状态", "phase", "条件"

**📍 代码位置**:
- 状态定义: `api/v1/nuwa_types.go:143-163` - `NuwaStatus` 结构
- 更新函数: `internal/controller/nuwa_controller.go:339-361` - `updateStatus`

**⚡ 修改指引**:
- 添加新状态字段: 在 `NuwaStatus` 添加字段，在 `updateStatus` 更新
- 添加 Condition: 使用 `meta.SetStatusCondition` 设置条件

---

### 部署运维 - 安装 CRD

**🔤 用户描述方式**:
- 主要: "安装 CRD", "部署 CRD", "注册资源"
- 别名: "install", "CRD 安装"

**📍 代码位置**:
- CRD 文件: `config/crd/bases/app.12306.work_nuwas.yaml`
- 安装命令: `Makefile:156-158` - `install` target

**⚡ 操作指引**:
```bash
make install      # 安装 CRD
make uninstall    # 卸载 CRD
```

---

### 部署运维 - 部署控制器

**🔤 用户描述方式**:
- 主要: "部署控制器", "安装 operator", "部署 manager"
- 别名: "deploy", "部署", "安装"

**📍 代码位置**:
- 部署配置: `config/manager/manager.yaml`
- 部署命令: `Makefile:166-168` - `deploy` target

**⚡ 操作指引**:
```bash
make deploy IMG=ghcr.io/ysicing/nuwa:latest
make undeploy
```

---

### 部署运维 - 本地开发

**🔤 用户描述方式**:
- 主要: "本地运行", "调试控制器", "开发模式"
- 别名: "run", "本地调试", "开发"

**📍 代码位置**:
- 入口文件: `cmd/main.go`
- 运行命令: `Makefile:112-113` - `run` target

**⚡ 操作指引**:
```bash
make run          # 本地运行（使用当前 kubeconfig）
make test         # 运行测试
make lint         # 代码检查
```

---

## 🔧 常见修改场景速查

| 用户需求 | 修改文件 | 关键位置 |
|---------|---------|---------|
| 添加新的 CRD 字段 | `api/v1/nuwa_types.go` | `NuwaSpec` 结构体 |
| 修改字段验证规则 | `api/v1/nuwa_types.go` | `+kubebuilder:validation` 标记 |
| 修改字段默认值 | `api/v1/nuwa_types.go` | `+kubebuilder:default` 标记 |
| 修改 CloneSet 生成逻辑 | `internal/controller/nuwa_controller.go` | `buildCloneSet` 函数 |
| 修改 Service 生成逻辑 | `internal/controller/nuwa_controller.go` | `buildService` 函数 |
| 修改状态更新逻辑 | `internal/controller/nuwa_controller.go` | `updateStatus` 函数 |
| 添加新的存储类型 | `api/v1/nuwa_types.go` + `nuwa_controller.go` | `StorageType` 枚举 + switch 分支 |
| 修改 RBAC 权限 | `internal/controller/nuwa_controller.go` | `+kubebuilder:rbac` 标记 |

---

## 📝 修改后必做

```bash
# 修改 *_types.go 后
make manifests    # 重新生成 CRD/RBAC
make generate     # 重新生成 DeepCopy

# 修改任何 .go 文件后
make build        # 编译验证
make test         # 运行测试
```
