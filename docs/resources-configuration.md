# Resources Configuration Guide

本文档说明 Nuwa CRD 中 `resources`、`resourcesPreset` 和 `resourcesOvercommit` 三个字段的关系和使用方法。

## 存储类型说明

Nuwa 支持三种存储类型，基于 OpenKruise CloneSet：

| 存储类型 | 适用场景 | 数据持久化 | 说明 |
|---------|---------|-----------|------|
| `PVC` | 需要持久化数据，但可随 Pod 删除 | **临时持久化** | CloneSet 的 PVC 会随 Pod 删除而删除，不同于 StatefulSet |
| `EmptyDir` | 临时数据、缓存 | 否 | Pod 删除后数据丢失 |
| `HostPath` | 需要访问宿主机文件系统 | 是（宿主机） | 数据保留在宿主机上 |

**重要**: CloneSet 使用 `volumeClaimTemplates` 创建的 PVC 会在 Pod 删除时一起删除。如果需要长期保留数据，建议：
- 使用 StatefulSet（不是 Nuwa 的范围）
- 或使用外部存储服务（如对象存储）

## 字段说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `resources` | `ResourceRequirements` | `nil` | 直接指定容器的资源请求和限制（Kubernetes 原生格式） |
| `resourcesPreset` | `enum` | `none` | 预设的资源配置模板（none/nano/micro/small/medium/large/xlarge/xxlarge） |
| `resourcesOvercommit` | `enum` | `none` | 资源超卖级别，控制 requests 与 limits 的比例（none/low/high） |

## 优先级规则

**核心规则**：`resources` > `resourcesPreset` + `resourcesOvercommit`

- 如果设置了 `resources`，则直接使用，忽略 `resourcesPreset` 和 `resourcesOvercommit`
- 如果未设置 `resources`，则根据 `resourcesPreset` 和 `resourcesOvercommit` 计算资源配置
- 如果 `resourcesPreset` 为 `none`（默认），则不应用任何资源限制

## 配置场景表

### 场景 1: 基础配置

| resources | resourcesPreset | resourcesOvercommit | 结果 | 说明 |
|-----------|----------------|---------------------|------|------|
| 未设置 | `none` (默认) | `none` (默认) | **无资源限制** | 默认行为，向后兼容现有 CR |
| 未设置 | `none` | `low` | **无资源限制** | preset 为 none 时，overcommit 被忽略 |
| 未设置 | `none` | `high` | **无资源限制** | preset 为 none 时，overcommit 被忽略 |

### 场景 2: 仅使用 Preset

| resources | resourcesPreset | resourcesOvercommit | CPU Limits | CPU Requests | Memory Limits | Memory Requests |
|-----------|----------------|---------------------|------------|--------------|---------------|-----------------|
| 未设置 | `nano` | `none` (默认) | 100m | 100m | 128Mi | 128Mi |
| 未设置 | `micro` | `none` | 250m | 250m | 256Mi | 256Mi |
| 未设置 | `small` | `none` | 500m | 500m | 512Mi | 512Mi |
| 未设置 | `medium` | `none` | 1 | 1 | 1Gi | 1Gi |
| 未设置 | `large` | `none` | 2 | 2 | 2Gi | 2Gi |
| 未设置 | `xlarge` | `none` | 4 | 4 | 4Gi | 4Gi |
| 未设置 | `xxlarge` | `none` | 8 | 8 | 8Gi | 8Gi |

### 场景 3: Preset + Overcommit 组合

| resources | resourcesPreset | resourcesOvercommit | CPU Limits | CPU Requests | Memory Limits | Memory Requests | 说明 |
|-----------|----------------|---------------------|------------|--------------|---------------|-----------------|------|
| 未设置 | `small` | `none` | 500m | 500m | 512Mi | 512Mi | 无超卖，requests = limits |
| 未设置 | `small` | `low` | 500m | **250m** | 512Mi | **256Mi** | 1x 超卖，requests = limits / 2 |
| 未设置 | `small` | `high` | 500m | **125m** | 512Mi | **128Mi** | 2x 超卖，requests = limits / 4 |
| 未设置 | `medium` | `none` | 1 | 1 | 1Gi | 1Gi | 无超卖 |
| 未设置 | `medium` | `low` | 1 | **500m** | 1Gi | **512Mi** | 1x 超卖 |
| 未设置 | `medium` | `high` | 1 | **250m** | 1Gi | **256Mi** | 2x 超卖 |

### 场景 4: 直接指定 Resources

| resources | resourcesPreset | resourcesOvercommit | 结果 | 说明 |
|-----------|----------------|---------------------|------|------|
| 已设置 | 任意值 | 任意值 | **使用 resources 的值** | resources 优先级最高 |
| `limits: {cpu: 2, memory: 2Gi}` | `small` | `low` | **使用 2 CPU / 2Gi** | preset 和 overcommit 被忽略 |
| `requests: {cpu: 100m}` | `medium` | `high` | **使用 100m CPU** | preset 和 overcommit 被忽略 |

## 超卖级别详解

| Overcommit Level | Divisor | Requests 计算公式 | 适用场景 |
|------------------|---------|------------------|----------|
| `none` | 1 | requests = limits | 生产环境，需要保证资源 |
| `low` | 2 | requests = limits / 2 | 开发/测试环境，适度超卖 |
| `high` | 4 | requests = limits / 4 | 资源紧张环境，高密度部署 |

## 使用建议

### 推荐配置

**生产环境**：
```yaml
spec:
  resourcesPreset: medium
  resourcesOvercommit: none  # 或不设置，默认为 none
```

**开发环境**：
```yaml
spec:
  resourcesPreset: small
  resourcesOvercommit: low
```

**高密度部署**：
```yaml
spec:
  resourcesPreset: small
  resourcesOvercommit: high
```

### 自定义资源配置

如果预设不满足需求，直接使用 `resources`：
```yaml
spec:
  resources:
    limits:
      cpu: "1.5"
      memory: "1.5Gi"
    requests:
      cpu: "500m"
      memory: "512Mi"
```

## 迁移指南

### 从无资源限制迁移

**现有 CR（无资源配置）**：
```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: my-app
spec:
  image: nginx:latest
  # 无 resources 配置
```

**行为**：
- 升级后：继续无资源限制（因为默认 `resourcesPreset: none`）
- 无需修改，向后兼容

**如需添加资源限制**：
```yaml
spec:
  image: nginx:latest
  resourcesPreset: small  # 显式设置
  resourcesOvercommit: low  # 可选
```

### 从 resources 迁移到 preset

**迁移前**：
```yaml
spec:
  resources:
    limits:
      cpu: "500m"
      memory: "512Mi"
    requests:
      cpu: "250m"
      memory: "256Mi"
```

**迁移后**（等效配置）：
```yaml
spec:
  resourcesPreset: small
  resourcesOvercommit: low
```

## QoS 类别对照

| 配置 | QoS Class | 说明 |
|------|-----------|------|
| `resourcesPreset: none` | BestEffort | 无资源限制 |
| `resourcesPreset: small, resourcesOvercommit: none` | Guaranteed | requests = limits |
| `resourcesPreset: small, resourcesOvercommit: low` | Burstable | requests < limits |
| `resourcesPreset: small, resourcesOvercommit: high` | Burstable | requests < limits |

## 常见问题

### Q1: 为什么默认是 none 而不是 small？

**A**: 为了向后兼容。现有的 CR 升级后不会自动应用资源限制，避免意外的滚动更新或调度约束。

### Q2: 可以只设置 resourcesOvercommit 吗？

**A**: 可以，但如果 `resourcesPreset` 为 `none`（默认），`resourcesOvercommit` 会被忽略。必须同时设置有效的 preset。

### Q3: resources 和 preset 可以同时设置吗？

**A**: 可以，但 `resources` 优先级更高，preset 和 overcommit 会被忽略。

### Q4: 如何查看实际应用的资源配置？

**A**: 查看生成的 CloneSet：
```bash
kubectl get cloneset <nuwa-name> -o yaml | grep -A 10 resources:
```

## 示例

### 示例 1: 微服务（低资源消耗）

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: api-gateway
spec:
  image: my-api:latest
  replicas: 3
  resourcesPreset: nano
  resourcesOvercommit: low
```

结果：
- CPU: limits=100m, requests=50m
- Memory: limits=128Mi, requests=64Mi

### 示例 2: Web 应用（中等资源）

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: web-app
spec:
  image: my-web:latest
  replicas: 2
  resourcesPreset: medium
  resourcesOvercommit: none
```

结果：
- CPU: limits=1, requests=1
- Memory: limits=1Gi, requests=1Gi

### 示例 3: 数据处理（高资源，自定义）

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: data-processor
spec:
  image: processor:latest
  replicas: 1
  resources:
    limits:
      cpu: "4"
      memory: "8Gi"
    requests:
      cpu: "2"
      memory: "4Gi"
```

结果：使用自定义的资源配置

### 示例 4: 向后兼容（无资源限制）

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: legacy-app
spec:
  image: legacy:latest
  # 不设置任何资源字段
```

结果：无资源限制（BestEffort QoS）
