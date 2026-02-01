# Storage Configuration Guide

本文档说明 Nuwa 中存储配置的使用方法和注意事项。

## 存储类型对比

Nuwa 支持三种存储类型：

| 存储类型 | 数据持久化 | Pod 删除后 | 适用场景 | 性能 | 多副本共享 |
|---------|-----------|-----------|---------|------|-----------|
| **PVC** | 是 | 数据保留 | 需要持久化数据的应用 | 中等（取决于 StorageClass） | 取决于 AccessMode |
| **EmptyDir** | 否 | 数据丢失 | 临时数据、缓存、构建产物 | 高（本地磁盘） | 否（每个 Pod 独立） |
| **HostPath** | 是（宿主机） | 数据保留在宿主机 | 日志收集、访问宿主机文件 | 高（本地磁盘） | 否（取决于调度） |

## PVC 存储说明

### 实现方式

Nuwa 为每个应用创建**一个独立的 PVC**（名称：`<nuwa-name>-data`），所有副本共享这个 PVC（如果 AccessMode 支持）。

**与 StatefulSet 的区别**：
- **StatefulSet**: 每个 Pod 有独立的 PVC（`pvc-0`, `pvc-1`, ...）
- **Nuwa**: 所有 Pod 共享一个 PVC（`<name>-data`）

### PVC 生命周期

```
Nuwa 创建 → PVC 创建 → 所有 Pod 挂载同一个 PVC
Nuwa 删除 → PVC 删除（通过 OwnerReference）
Pod 删除 → PVC 保留（不受影响）
```

### AccessMode 选择

| AccessMode | 说明 | 多副本支持 | 适用场景 |
|-----------|------|-----------|---------|
| `ReadWriteOnce` (默认) | 单节点读写 | ❌ 仅支持单副本或同节点调度 | 单副本应用 |
| `ReadOnlyMany` | 多节点只读 | ✅ 支持多副本 | 静态资源、配置文件 |
| `ReadWriteMany` | 多节点读写 | ✅ 支持多副本 | 需要 NFS 等支持的 StorageClass |

**重要**: 使用 `ReadWriteOnce` 时，多副本应用需要确保所有 Pod 调度到同一节点，否则会出现挂载失败。

## 配置示例

### 1. PVC 存储（持久化存储）

适用场景：
- 需要持久化数据的应用
- 数据需要在 Pod 重启后保留
- 多副本应用需要共享数据（需要 ReadWriteMany）

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: app-with-pvc
spec:
  image: myapp:latest
  replicas: 2
  storage:
    type: PVC
    mountPath: /data
    size: 10Gi
    storageClassName: standard
    accessModes:
      - ReadWriteOnce
```

**注意事项**：
- PVC 独立于 Pod，不会随 Pod 删除而删除
- 所有副本共享同一个 PVC（名称：`<nuwa-name>-data`）
- 使用 `ReadWriteOnce` 时，多副本需要调度到同一节点
- 使用 `ReadWriteMany` 可支持多节点多副本（需要 StorageClass 支持）

### 2. EmptyDir 存储（临时数据）

适用场景：
- 临时文件、缓存
- 构建产物
- 不需要持久化的数据

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: app-with-emptydir
spec:
  image: myapp:latest
  replicas: 2
  storage:
    type: EmptyDir
    mountPath: /tmp/cache
    size: 5Gi  # 可选，限制 EmptyDir 大小
```

**注意事项**：
- Pod 删除后数据立即丢失
- 性能最好（使用节点本地磁盘）
- 可选设置 `size` 限制大小

### 3. HostPath 存储（宿主机路径）

适用场景：
- 日志收集
- 访问宿主机配置文件
- 需要在 Pod 重建后保留数据

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: app-with-hostpath
spec:
  image: myapp:latest
  replicas: 1  # 注意：HostPath 通常只适合单副本
  storage:
    type: HostPath
    mountPath: /app/logs
    hostPath: /var/log/myapp
    hostPathType: DirectoryOrCreate
```

**注意事项**：
- 数据保留在宿主机上
- 多副本时，不同 Pod 可能在不同节点，访问不同的 hostPath
- 需要注意安全性和权限问题

## 存储大小配置

### 默认值

| 存储类型 | 默认大小 | 说明 |
|---------|---------|------|
| PVC | `8Gi` | 在控制器中应用默认值 |
| EmptyDir | 无限制 | 不设置 `size` 时无大小限制 |
| HostPath | N/A | 使用宿主机磁盘空间 |

### 自定义大小

```yaml
spec:
  storage:
    type: PVC
    size: 20Gi  # 自定义大小
```

## PVC 状态检查

Nuwa 会自动检查 PVC 状态并更新到 `.status.pvcStatus`：

```yaml
status:
  pvcStatus: Bound  # 或 Pending, NotFound
  conditions:
    - type: PVCReady
      status: "True"
      reason: PVCBound
      message: "All 2 PVCs are bound"
```

**状态说明**：
- `Bound`: 所有 PVC 已绑定
- `Pending`: 部分 PVC 未绑定
- `NotFound`: 未找到 PVC

## 最佳实践

### 1. 选择合适的存储类型

| 需求 | 推荐类型 | 原因 |
|------|---------|------|
| 临时缓存 | EmptyDir | 性能最好，无需持久化 |
| 应用日志（短期） | PVC | 可持久化，便于调试 |
| 应用日志（长期） | HostPath 或外部日志系统 | 数据保留在节点或外部系统 |
| 共享配置文件 | PVC (ReadOnlyMany) | 多副本共享只读数据 |
| 共享数据（读写） | PVC (ReadWriteMany) | 需要支持 RWX 的 StorageClass |
| 数据库数据 | **不推荐使用 Nuwa** | 使用 StatefulSet 或托管数据库 |
| 用户上传文件 | **不推荐使用 Nuwa** | 使用对象存储（如 S3） |

### 2. 避免数据丢失

**错误示例**（数据会丢失）：
```yaml
# ❌ 不要这样做：用 PVC 存储重要数据
spec:
  image: myapp:latest
  storage:
    type: PVC
    mountPath: /data/important  # Pod 删除后数据丢失！
```

**正确做法**：
```yaml
# ✅ 使用外部存储
spec:
  image: myapp:latest
  env:
    - name: S3_BUCKET
      value: my-app-data
    - name: S3_ENDPOINT
      value: https://s3.amazonaws.com
  # 应用代码将数据存储到 S3
```

### 3. 多副本注意事项

**PVC 类型**：
- 每个 Pod 有独立的 PVC
- 不同 Pod 之间数据不共享
- 适合无状态应用

**HostPath 类型**：
- 不同 Pod 可能在不同节点
- 访问不同的 hostPath
- 通常只适合单副本或使用节点亲和性

**EmptyDir 类型**：
- 每个 Pod 有独立的 EmptyDir
- 不同 Pod 之间数据不共享
- 适合无状态应用

## 迁移指南

### 从无存储迁移到 PVC

```yaml
# 迁移前
spec:
  image: myapp:latest

# 迁移后
spec:
  image: myapp:latest
  storage:
    type: PVC
    mountPath: /data
    size: 10Gi
```

**注意**: 添加存储配置会触发滚动更新。

### 从 EmptyDir 迁移到 PVC

```yaml
# 迁移前
spec:
  storage:
    type: EmptyDir
    mountPath: /tmp/cache

# 迁移后
spec:
  storage:
    type: PVC
    mountPath: /tmp/cache
    size: 5Gi
```

**注意**:
- 迁移会触发滚动更新
- EmptyDir 中的数据会丢失
- 新 PVC 会被创建

## 常见问题

### Q1: PVC 什么时候会被删除？

**A**: PVC 通过 OwnerReference 与 Nuwa 资源绑定，只有在删除 Nuwa 资源时才会被删除。Pod 删除或重启不会影响 PVC。

### Q2: 如何在 Pod 重建后恢复数据？

**A**: 数据会自动恢复。因为 PVC 独立于 Pod，Pod 重建后会自动挂载同一个 PVC，数据保持不变。

### Q3: 多副本应用如何共享数据？

**A**: Nuwa 的存储类型都不支持多 Pod 共享。建议：
- 使用 ReadWriteMany 的 PVC（需要支持的 StorageClass）
- 使用外部存储服务（S3、Redis、数据库）
- 使用 NFS 或其他共享文件系统

### Q4: EmptyDir 的 size 限制是硬限制吗？

**A**: 是的。设置 `size: 5Gi` 后，EmptyDir 最多使用 5Gi 空间。超过限制会导致 Pod 被驱逐。

### Q5: 如何查看 PVC 状态？

**A**:
```bash
# 查看 Nuwa 状态
kubectl get nuwa my-app -o yaml | grep -A 5 pvcStatus

# 查看 PVC 列表
kubectl get pvc -l app.kubernetes.io/name=my-app

# 查看 PVC 详情
kubectl describe pvc data-my-app-xxxxx
```

## 示例

### 示例 1: Web 应用（临时缓存）

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: web-app
spec:
  image: nginx:latest
  replicas: 3
  storage:
    type: EmptyDir
    mountPath: /var/cache/nginx
    size: 1Gi
```

### 示例 2: 日志收集（HostPath）

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: log-collector
spec:
  image: fluentd:latest
  replicas: 1
  storage:
    type: HostPath
    mountPath: /var/log/app
    hostPath: /var/log/containers
    hostPathType: Directory
```

### 示例 3: 构建服务（PVC）

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: build-service
spec:
  image: builder:latest
  replicas: 2
  storage:
    type: PVC
    mountPath: /workspace
    size: 20Gi
    storageClassName: fast-ssd
```

## 总结

- **PVC**: 持久化存储，数据保留，所有副本共享（取决于 AccessMode）
- **EmptyDir**: 临时数据，性能最好，每个 Pod 独立
- **HostPath**: 宿主机路径，数据保留在节点上

**关键原则**:
- PVC 适合需要持久化的无状态应用
- 需要每个副本独立存储的场景，使用 StatefulSet
- 需要长期保留且高可用的数据，使用外部存储服务
