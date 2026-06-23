# Redis Operator Helm Chart

一个用于管理 Redis 实例的 Kubernetes Operator。

## 安装

### 添加 Helm 仓库（如果有 Chart 仓库的话）

```bash
helm repo add bubua12 https://bubua12.github.io/charts
helm repo update
```

### 本地安装

```bash
# 从本地 Chart 目录安装
helm install redis-operator ./deploy/helm/redis-operator \
  --namespace redis-operator-system \
  --create-namespace
```

### 自定义配置安装

```bash
helm install redis-operator ./deploy/helm/redis-operator \
  --namespace redis-operator-system \
  --create-namespace \
  --set operator.image.tag=v0.1.0 \
  --set operator.replicas=1
```

## 配置参数

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `operator.image.repository` | 镜像仓库 | `bubua12/redis-operator` |
| `operator.image.tag` | 镜像标签 | `v0.0.1` |
| `operator.image.pullPolicy` | 拉取策略 | `IfNotPresent` |
| `operator.replicas` | Operator 副本数 | `1` |
| `operator.resources.limits.cpu` | CPU 限制 | `500m` |
| `operator.resources.limits.memory` | 内存限制 | `128Mi` |
| `operator.resources.requests.cpu` | CPU 请求 | `10m` |
| `operator.resources.requests.memory` | 内存请求 | `64Mi` |
| `operator.leaderElect` | 启用 Leader 选举 | `true` |
| `namespace.create` | 是否创建命名空间 | `true` |
| `namespace.name` | 命名空间名称 | `redis-operator-system` |
| `rbac.create` | 是否创建 RBAC 资源 | `true` |
| `serviceAccount.create` | 是否创建 ServiceAccount | `true` |
| `crd.create` | 是否安装 CRD | `true` |

## 使用示例

### 创建 Redis 实例

```yaml
apiVersion: cache.bubua12.com/v1alpha1
kind: Redis
metadata:
  name: my-redis
  namespace: default
spec:
  replicas: 3
  image: redis:7.2-alpine
  port: 6379
```

### 查看 Redis 状态

```bash
kubectl get redis my-redis -o yaml
```

Status 会显示：
```yaml
status:
  readyReplicas: 3
  state: Running
```

## 卸载

```bash
# 卸载 Operator
helm uninstall redis-operator -n redis-operator-system

# 卸载 CRD（可选，会删除所有 Redis 实例）
kubectl delete crd redis.cache.bubua12.com
```

## 开发

### 本地测试

```bash
# 验证 Chart 语法
helm template redis-operator ./deploy/helm/redis-operator

# 本地 dry-run
helm install redis-operator ./deploy/helm/redis-operator \
  --namespace redis-operator-system \
  --create-namespace \
  --dry-run
```

### 打包发布

```bash
# 打包 Chart
helm package ./deploy/helm/redis-operator

# 推送到 Chart 仓库（需要配置）
helm push redis-operator-0.1.0.tgz oci://registry-1.docker.io/bubua12
```

## 相关链接

- [GitHub 仓库](https://github.com/bubua12/bubua12-redis-operator)
- [学习指南](docs/LEARNING_GUIDE.md)
- [产品路线图](docs/ROADMAP.md)
