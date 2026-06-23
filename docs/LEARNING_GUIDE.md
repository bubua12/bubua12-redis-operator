# Redis Operator 从零学习导航

> 目标：从零开始，逐步构建一个 Redis Operator，理解 CRD、Controller、Reconcile 模式，
> 以及如何用 YAML 部署到 Kubernetes。

---

## 学习路线总览

```
阶段 0: 基础知识       ──→  理论铺垫
阶段 0.5: 环境搭建     ──→  ✅ 已完成（Go + kubebuilder + 脚手架）
阶段 1: CRD 定义       ──→  定义 Redis 集群长什么样（⬅ 你在这里）
阶段 2: Controller     ──→  告诉 K8s 怎么把 Redis 跑起来
阶段 3: 部署 YAML      ──→  把 Operator 自己装进 K8s
阶段 4: 高级功能       ──→  自动扩缩容、故障转移、监控
```

---

## 阶段 0: 基础知识（先读再写）

### 0.1 Kubernetes 核心概念回顾

| 概念 | 你需要知道的 | 为什么重要 |
|------|-------------|-----------|
| Pod | K8s 最小调度单元 | Redis 实例最终跑在 Pod 里 |
| StatefulSet | 有状态应用的控制器 | Redis 集群需要稳定的网络标识和持久存储 |
| Service | 网络抽象 | 客户端通过 Service 连接 Redis |
| ConfigMap / Secret | 配置管理 | 存储 redis.conf、密码等 |
| PVC (PersistentVolumeClaim) | 持久化存储 | Redis 数据落盘 |
| RBAC | 权限控制 | Operator 需要权限操作这些资源 |

### 0.2 Operator 模式核心概念

```
┌─────────────────────────────────────────────────────┐
│                    Kubernetes API                     │
│                                                      │
│   用户提交 YAML  ──→  CRD (声明期望状态)              │
│                           │                          │
│                      Controller 监听                  │
│                           │                          │
│                    ┌──────┴──────┐                    │
│                    │  Reconcile  │  ← 核心循环         │
│                    │  期望 vs 实际 │                    │
│                    └──────┬──────┘                    │
│                           │                          │
│              创建/更新 Pods, Services, ConfigMaps...   │
└─────────────────────────────────────────────────────┘
```

**关键术语：**
- **CRD (Custom Resource Definition)**：告诉 K8s "有一种新资源叫 RedisCluster"
- **Controller**：监听 CRD 变化，执行实际操作的代码
- **Reconcile**：核心循环 — 不断比较"期望状态"和"实际状态"，驱动实际趋近期望
- **Operator = CRD + Controller**：用代码封装运维知识

### 0.3 推荐阅读

- [ ] [Kubernetes 官方文档 - Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [ ] [Operator 模式](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [ ] [controller-runtime 项目](https://github.com/kubernetes-sigs/controller-runtime)
- [ ] [Kubebuilder 官方教程](https://book.kubebuilder.io/)（我们的主要参考）

---

## 阶段 0.5: 环境搭建与项目初始化（✅ 已完成）

> 实际执行记录，供后续回顾参考。

### 环境信息

| 项目 | 版本 |
|------|------|
| OS | Windows 11 + WSL2 |
| WSL 发行版 | Ubuntu 26.04 LTS |
| Go | 1.25.11 linux/amd64 |
| kubebuilder | v4.13.0 |

### Go 安装（WSL 内）

```bash
# 下载
curl -fsSL https://go.dev/dl/go1.25.11.linux-amd64.tar.gz -o /tmp/go1.25.11.linux-amd64.tar.gz

# 解压到用户目录（不需要 sudo）
mkdir -p ~/.local
rm -rf ~/.local/go
tar -C ~/.local -xzf /tmp/go1.25.11.linux-amd64.tar.gz

# 配置 PATH（写入 ~/.profile，因为 .bashrc 非交互式 shell 会 return）
# 在 ~/.profile 末尾追加：
#   export PATH="$HOME/.local/go/bin:$PATH"
#   export GOPATH="$HOME/go"
#   export PATH="$PATH:$GOPATH/bin"

# 生效
source ~/.profile
go version   # → go version go1.25.11 linux/amd64
```

### kubebuilder 安装

```bash
go install sigs.k8s.io/kubebuilder/v4@v4.13.0

# ⚠️ 如果 /usr/local/bin 下有旧的坏文件，先删掉：
# sudo rm /usr/local/bin/kubebuilder

kubebuilder version   # → KubeBuilder: v4.13.0
```

### 项目初始化

```bash
cd /mnt/d/workspaces/GolandProjects/bubua12-redis-operator

# 第一步：初始化项目骨架
kubebuilder init \
  --domain bubua12.com \
  --repo github.com/bubua12/bubua12-redis-operator \
  --project-name redis-operator

# 第二步：创建 Redis CRD + Controller 脚手架
kubebuilder create api \
  --group cache \
  --version v1alpha1 \
  --kind Redis

# 第三步：安装 make（WSL 默认没有）
sudo apt update && sudo apt install -y make

# 第四步：生成 deepcopy 等代码
make generate
```

### 生成的关键文件

```
api/v1alpha1/redis_types.go                ← CRD 类型定义（下一步要改的）
api/v1alpha1/groupversion_info.go          ← 版本注册，不用动
api/v1alpha1/zz_generated.deepcopy.go      ← 自动生成的深拷贝
internal/controller/redis_controller.go    ← 控制器逻辑（后面要写的）
internal/controller/redis_controller_test.go
internal/controller/suite_test.go
cmd/main.go                                ← 入口，不用改
config/                                    ← K8s 部署 YAML，自动生成
```

### 项目目录结构（实际）

```
bubua12-redis-operator/
├── cmd/main.go                         ← 入口文件
├── Makefile                            ← 构建命令
├── PROJECT                             ← kubebuilder 元数据
├── Dockerfile                          ← 镜像构建
├── api/
│   └── v1alpha1/
│       ├── redis_types.go              ← 📌 CRD 定义，要手写
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go
├── internal/
│   └── controller/
│       ├── redis_controller.go         ← 📌 控制器逻辑，要手写
│       ├── redis_controller_test.go
│       └── suite_test.go
├── config/
│   ├── crd/                            ← CRD YAML
│   ├── rbac/                           ← RBAC 权限
│   ├── manager/                        ← Operator Deployment
│   ├── default/                        ← Kustomize 入口
│   ├── prometheus/                     ← 监控配置
│   └── network-policy/
├── test/
├── hack/
└── docs/
    └── LEARNING_GUIDE.md               ← 你正在读的文件
```

---

## 阶段 1: 定义 CRD — "Redis 集群长什么样"

> 文件: `api/v1alpha1/redis_types.go`

### 1.1 用户视角 — 最终的 YAML

用户希望这样声明一个 Redis 集群：

```yaml
apiVersion: cache.bubua12.com/v1alpha1
kind: Redis
metadata:
  name: my-redis
  namespace: default
spec:
  replicas: 3              # 3 个 Redis 实例
  image: redis:7.2-alpine  # 使用的镜像
  port: 6379               # 服务端口
  passwordSecretRef:       # 引用密码 Secret
    name: my-redis-password
  storage:
    size: 10Gi             # 存储容量
    storageClassName: standard
  resources:               # 资源限制
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
```

### 1.2 对应的 Go 结构体（Spec 和 Status）

```
api/
└── v1alpha1/
    ├── redis_types.go            ← CRD 类型定义（核心）
    ├── groupversion_info.go      ← API Group 版本注册
    └── zz_generated.deepcopy.go  ← 自动生成的深拷贝方法
```

**要理解的概念：**
- `Spec` — 用户想要什么（声明式，期望状态）
- `Status` — 系统当前是什么（由 Controller 更新）
- `+kubebuilder:object:root=true` — 标记为 K8s 标准资源
- `+kubebuilder:subresource:status` — 启用 status 子资源

### 1.3 学习步骤

```
第 1 步: 初始化项目（✅ 已完成，见阶段 0.5）
  → go mod init, 安装 kubebuilder
  → kubebuilder init --domain bubua12.com
  → kubebuilder create api --group cache --version v1alpha1 --kind Redis

第 2 步: 定义 Spec 字段
  → 编辑 redis_types.go
  → 添加 Image, Replicas, Port, Storage 等字段

第 3 步: 定义 Status 字段
  → ReadyReplicas: 实际就绪的实例数
  → State: 当前状态 (Creating/Running/Failed)
  → Conditions: 详细的状态条件列表

第 4 步: 生成 CRD YAML
  → make generate && make manifests
  → 查看生成的 config/crd/bases/ 下的 YAML

第 5 步: 理解生成的文件
  → 每个文件的作用是什么
  → deepcopy 为什么需要
  → groupversion_info 注册了什么
```

### 1.4 产出物

| 文件 | 作用 |
|------|------|
| `api/v1alpha1/redis_types.go` | Go 结构体定义 |
| `config/crd/bases/cache.bubua12.com_redises.yaml` | 生成的 CRD YAML |
| `api/v1alpha1/zz_generated.deepcopy.go` | 自动生成的深拷贝代码 |

---

## 阶段 2: 编写 Controller — "怎么把 Redis 跑起来"

> 文件: `internal/controller/redis_controller.go`

### 2.1 Reconcile 的工作流

```
Reconcile(ctx, req) 被调用
    │
    ├─ 1. 获取 RedisCluster 对象 (Get)
    │     → 如果不存在，说明被删了，清理并返回
    │
    ├─ 2. 查看当前状态，决定操作
    │     → 是否已有 StatefulSet？需要更新还是新建？
    │     → 是否已有 Service？ConfigMap？
    │
    ├─ 3. 确保子资源存在且正确 (Upsert)
    │     → ConfigMap  (redis.conf 配置)
    │     → Service    (headless service for StatefulSet)
    │     → StatefulSet (Redis 实例)
    │     → Password Secret (如果没有指定则自动生成)
    │
    ├─ 4. 检查 StatefulSet 状态
    │     → ReadyReplicas 是否等于期望 replicas？
    │     → 更新 RedisCluster.Status
    │
    └─ 5. 返回
          → err != nil  → 自动重试
          → return ctrl.Result{RequeueAfter: 30s} → 30秒后再检查
```

### 2.2 需要创建/管理的 K8s 资源

```
RedisCluster CR
    │
    ├──→ ConfigMap (redis.conf)
    │     └── 配置: maxmemory, appendonly, requirepass 等
    │
    ├──→ Secret (密码)
    │     └── 自动生成随机密码，或引用用户指定的
    │
    ├──→ Headless Service
    │     └── name: my-redis-headless
    │     └── 用于 StatefulSet 的 DNS 解析
    │
    ├──→ Client Service
    │     └── name: my-redis
    │     └── 用于客户端连接
    │
    └──→ StatefulSet
          ├── replicas: spec.replicas
          ├── template:
          │     └── containers: [redis]
          │           └── 使用 ConfigMap 挂载配置
          │           └── 使用 Secret 挂载密码
          └── volumeClaimTemplates: [数据盘]
```

### 2.3 学习步骤

```
第 1 步: 理解 Reconcile 函数签名
  → func (r *RedisReconciler) Reconcile(ctx, req) (Result, error)
  → 什么是 ctrl.Result？Requeue 和 RequeueAfter 的区别

第 2 步: 实现 Get — 获取 RedisCluster 对象
  → client.Get(ctx, req.NamespacedName, redis)
  → 处理 NotFound 错误

第 3 步: 实现 ConfigMap
  → 构建 redis.conf 内容
  → 使用 ctrl.SetControllerReference 设置 OwnerReference
  → 创建或更新 ConfigMap

第 4 步: 实现 Headless Service
  → ClusterIP: None (headless)
  → 设置 OwnerReference

第 5 步: 实现 StatefulSet
  → 这是最复杂的部分
  → 配置容器模板、环境变量、卷挂载
  → 确保 replicas 和 image 正确

第 6 步: 实现 Status 更新
  → 观察 StatefulSet.Status
  → 更新 RedisCluster.Status.ReadyReplicas

第 7 步: SetupWithManager
  → 注册 Watches: 监听哪些资源的变化触发 Reconcile
  → Owns(&appsv1.StatefulSet{}): 子资源变化时触发
```

### 2.4 关键代码模式

```go
// OwnerReference 模式 — 子资源跟随父资源自动清理
ctrl.SetControllerReference(owner, child, r.Scheme)

// 创建或更新 (Upsert) 模式
existing := &corev1.ConfigMap{}
err := r.Get(ctx, key, existing)
if errors.IsNotFound(err) {
    return r.Create(ctx, desired)  // 不存在就创建
} else if err != nil {
    return err
}
return r.Update(ctx, existing)     // 存在就更新

// Requeue 控制
return ctrl.Result{RequeueAfter: 30 * time.Second}, nil  // 30秒后再检查
```

---

## 阶段 3: 部署 YAML — "把 Operator 装进 K8s"

### 3.1 部署架构

```
namespace: redis-operator-system
    │
    ├── Deployment (Operator 本身)
    │     └── 1 个 Pod 跑 Operator 进程
    │
    ├── ServiceAccount
    │     └── Operator 使用的账号
    │
    ├── ClusterRole
    │     └── Operator 需要的权限列表
    │
    ├── ClusterRoleBinding
    │     └── 把 Role 绑定到 ServiceAccount
    │
    ├── CRD (阶段 1 生成的)
    │     └── 告诉 K8s RedisCluster 是一种合法资源
    │
    └── (可选) Webhook / CertManager 配置
```

### 3.2 关键 YAML 文件说明

```
config/
├── crd/
│   └── bases/
│       └── cache.bubua12.com_redises.yaml     ← CRD 定义
├── rbac/
│   ├── role.yaml                                     ← 权限定义
│   ├── role_binding.yaml                             ← 权限绑定
│   └── service_account.yaml                          ← 服务账号
├── manager/
│   └── manager.yaml                                  ← Operator Deployment
├── default/
│   └── kustomization.yaml                            ← Kustomize 入口
└── samples/
    └── redis_v1_rediscluster.yaml                    ← 示例 CR
```

### 3.3 学习步骤

```
第 1 步: 用 kubebuilder 生成默认配置
  → make manifests 生成 config/ 下的 YAML
  → 逐个文件阅读，理解每个 YAML 的作用

第 2 步: 理解 RBAC
  → Operator 需要哪些权限？
  → 为什么需要 ClusterRole 而不是 Role？
  → +kubebuilder:rbac 注解如何映射到 role.yaml

第 3 步: 本地测试运行 (无需部署到集群)
  → make install          → 安装 CRD 到集群
  → make run              → 本地运行 Operator
  → kubectl apply -f config/samples/ → 创建示例 CR

第 4 步: 构建并推送镜像
  → make docker-build IMG=<你的仓库>/redis-operator:v0.0.1
  → make docker-push  IMG=<你的仓库>/redis-operator:v0.0.1

第 5 步: 部署到集群
  → make deploy IMG=<你的仓库>/redis-operator:v0.0.1
  → 观察 Operator Pod 启动
  → 再次 apply 示例 CR，观察 Redis 实例被创建

第 6 步: 理解 Kustomize
  → config/default/kustomization.yaml 如何组合各层
  → 如何覆盖 namespace、image 等
```

---

## 阶段 4: 测试 — "确保它真的能用"

### 4.1 测试金字塔

```
        /  E2E Tests  \         ← 需要真实集群 (少而精)
       /  Envtest Tests \       ← 用 envtest 模拟 API Server (推荐)
      /   Unit Tests     \      ← 纯函数逻辑测试 (快)
```

### 4.2 学习步骤

```
第 1 步: Unit Tests
  → 测试辅助函数 (构建 ConfigMap 数据、生成密码等)
  → 不需要 K8s 环境

第 2 步: Envtest Tests (重点)
  → 用 envtest 启动真实的 etcd + API Server (轻量)
  → 完整测试 Reconcile 逻辑
  → 验证: 创建 CR → 检查 StatefulSet 是否被创建
  → 验证: 修改 replicas → 检查 StatefulSet 是否更新

第 3 步: E2E Tests (可选)
  → 需要真实 K8s 集群
  → 用 Kind 搭建测试集群
  → 完整的部署流程测试
```

---

## 阶段 5: 高级功能（可选，进阶）

| 功能 | 描述 | 关键技术 |
|------|------|---------|
| **自动故障转移** | 主节点挂了自动提升从节点 | sentinel 或 redis-cluster 模式 |
| **水平扩缩容** | 改 spec.replicas 自动扩缩 | StatefulSet scale |
| **滚动更新** | 升级 Redis 版本不中断服务 | StatefulSet updateStrategy |
| **Webhook 校验** | 拒绝不合法的 CR | Admission Webhook |
| **Metrics 导出** | 暴露 Operator 指标 | controller-runtime metrics |
| **Prometheus 监控** | Redis 指标采集 | sidecar exporter |
| **Backup/Restore** | 数据备份恢复 | CronJob + RDB/AOF |

---

## 推荐的动手顺序

```
✅ [1] 阅读阶段 0 的基础概念 ────→ 已完成
✅ [2] 环境搭建 + kubebuilder 初始化 ──→ 已完成（见阶段 0.5）
    │
    ▼
[3] 完善 CRD (阶段 1) ────→ ⬅ 你在这里，1-2 小时
    │   → 编辑 redis_types.go，定义 Spec 和 Status
    │   → make generate && make manifests
    │   → 理解生成的 CRD YAML
    │
    ▼
[4] 编写 Controller (阶段 2) ────→ 这是重点，3-5 小时
    │   → 从最简单开始: 只管一个 StatefulSet
    │   → make run 本地运行测试
    │   → 逐步添加 ConfigMap, Service
    │
    ▼
[5] 部署到集群 (阶段 3) ────→ 1-2 小时
    │   → make docker-build && make deploy
    │   → 端到端验证
    │
    ▼
[6] 写测试 (阶段 4) ────→ 2-3 小时
    │
    ▼
[7] 高级功能 (阶段 5) ────→ 按需学习
```

---

## 项目目录结构预览

> ⬇️ 以下为实际生成的结构（group=cache, version=v1alpha1, kind=Redis）

```
bubua12-redis-operator/
├── go.mod / go.sum
├── Makefile                        ← kubebuilder 生成的构建脚本
├── PROJECT                         ← kubebuilder 元数据
├── Dockerfile                      ← Operator 镜像构建
├── api/
│   └── v1alpha1/
│       ├── redis_types.go          ← [阶段1] CRD 类型定义
│       ├── groupversion_info.go    ← [阶段1] 版本注册
│       └── zz_generated.deepcopy.go
├── internal/
│   └── controller/
│       ├── redis_controller.go             ← [阶段2] 核心控制器
│       ├── redis_controller_test.go        ← [阶段4] 测试
│       └── suite_test.go
├── cmd/
│   └── main.go                     ← 入口: 启动 Manager
├── config/
│   ├── crd/                        ← [阶段1] CRD YAML
│   ├── rbac/                       ← [阶段3] RBAC 配置
│   ├── manager/                    ← [阶段3] Deployment
│   ├── default/                    ← [阶段3] Kustomize 入口
│   └── samples/                    ← 示例 CR
└── docs/
    └── LEARNING_GUIDE.md           ← 你正在读的文件
```

---

## 常用命令速查

| 命令 | 作用 |
|------|------|
| `kubebuilder init --domain example.com` | 初始化项目 |
| `kubebuilder create api --group cache --version v1alpha1 --kind Redis` | 创建 CRD + Controller 脚手架 |
| `make generate` | 运行代码生成器 (deepcopy 等) |
| `make manifests` | 生成 CRD/RBAC YAML |
| `make install` | 把 CRD 安装到当前集群 |
| `make uninstall` | 从集群删除 CRD |
| `make run` | 本地运行 Operator (连远程集群) |
| `make docker-build IMG=xxx` | 构建 Operator 镜像 |
| `make deploy IMG=xxx` | 部署 Operator 到集群 |
| `make undeploy` | 从集群删除 Operator |

---

> 💡 **学习建议**: 不要试图一次理解所有东西。按阶段推进，每个阶段先动手做，
> 遇到不理解的再深入看。Kubernetes Operator 的核心就是 **Reconcile 循环** —
> 一旦你理解了"期望状态 → 对比实际 → 执行操作"这个模式，其他都是细节。
