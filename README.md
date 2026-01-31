# Nuwa

Nuwa 是一个 Kubernetes CRD 控制器，基于 [OpenKruise](https://openkruise.io/) CloneSet 实现简化的应用部署。

只需配置镜像、环境变量、端口、存储等基本信息，Nuwa 会自动创建 CloneSet 和 Service。

## 功能特性

- 简化配置：只需定义核心字段，无需编写复杂的 YAML
- 自动创建 Service：配置端口后自动创建对应的 Service
- 多种存储类型：支持 PVC、EmptyDir、HostPath
- 基于 OpenKruise：享受原地升级、分批发布等高级特性

## 前置条件

- Kubernetes v1.20+
- [OpenKruise](https://openkruise.io/docs/installation) 已安装
- kubectl 已配置

## 快速开始

### 安装 OpenKruise

```bash
helm repo add openkruise https://openkruise.github.io/charts/
helm install kruise openkruise/kruise --version 1.8.0
```

### 安装 Nuwa

```bash
# 安装 CRD
make install

# 部署控制器
make deploy IMG=ghcr.io/ysicing/nuwa:latest
```

### 创建应用

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: my-app
spec:
  image: nginx:latest
  replicas: 2
  env:
    - name: TZ
      value: Asia/Shanghai
  ports:
    - containerPort: 80
```

```bash
kubectl apply -f my-app.yaml
```

## 配置说明

### 完整示例

```yaml
apiVersion: app.12306.work/v1
kind: Nuwa
metadata:
  name: my-app
spec:
  # 镜像（必填）
  image: nginx:latest

  # 副本数（默认 1）
  replicas: 2

  # 环境变量
  env:
    - name: TZ
      value: Asia/Shanghai
    - name: DB_HOST
      valueFrom:
        secretKeyRef:
          name: db-secret
          key: host

  # 端口映射（配置后自动创建 Service）
  ports:
    - name: http
      containerPort: 80
      servicePort: 80
    - name: metrics
      containerPort: 9090

  # Service 类型：ClusterIP（默认）、NodePort、LoadBalancer
  serviceType: ClusterIP

  # 资源限制
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

  # 存储配置
  storage:
    type: PVC          # PVC、EmptyDir、HostPath
    size: 10Gi
    mountPath: /data
    storageClassName: standard

  # 镜像拉取策略
  imagePullPolicy: IfNotPresent

  # 私有镜像仓库凭证
  imagePullSecrets:
    - name: registry-secret
```

### 存储类型

| 类型 | 说明 | 示例 |
|------|------|------|
| `PVC` | 持久卷声明（默认） | 数据库、文件存储 |
| `EmptyDir` | 临时存储，Pod 删除后数据丢失 | 缓存、临时文件 |
| `HostPath` | 主机路径挂载 | 日志收集、特殊场景 |

**PVC 示例：**
```yaml
storage:
  type: PVC
  size: 10Gi
  mountPath: /data
  storageClassName: standard
  accessModes:
    - ReadWriteOnce
```

**EmptyDir 示例：**
```yaml
storage:
  type: EmptyDir
  mountPath: /tmp/cache
  size: 1Gi  # 可选，限制大小
```

**HostPath 示例：**
```yaml
storage:
  type: HostPath
  mountPath: /data
  hostPath: /mnt/data
  hostPathType: DirectoryOrCreate
```

## 状态查看

```bash
# 查看 Nuwa 资源
kubectl get nuwa

# 输出示例
NAME       IMAGE          REPLICAS   READY   PHASE     AGE
my-app     nginx:latest   2          2       Running   5m
```

## 卸载

```bash
# 删除应用
kubectl delete nuwa my-app

# 卸载控制器
make undeploy

# 删除 CRD
make uninstall
```

## 开发

```bash
# 本地运行（开发调试）
make run

# 运行测试
make test

# 构建镜像
make docker-build IMG=ghcr.io/ysicing/nuwa:latest

# 查看所有可用命令
make help
```

## License

Apache License 2.0
