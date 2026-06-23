# Redis Operator 产品路线图 (Roadmap)

> 最后更新：2026-06-23
> 当前版本：v0.0.1
> 状态：基础功能已完成，进入进阶功能规划阶段

---

## 产品愿景

打造一个**生产可用**的 Redis Operator，让用户通过简单的 YAML 声明即可在 Kubernetes 上部署和管理 Redis 集群，无需关心底层实现细节。

---

## 已完成（v0.0.1）

| 功能 | 状态 | 说明 |
|------|------|------|
| CRD 定义 | ✅ | 支持 Replicas / Image / Port 字段 |
| StatefulSet 管理 | ✅ | 自动创建/更新 Redis Pod |
| Service 管理 | ✅ | 自动创建 ClusterIP 类型 Service |
| Status 状态同步 | ✅ | 实时更新 ReadyReplicas 和 State |
| 多命名空间支持 | ✅ | 跟随 Redis CR 的命名空间自动部署 |
| 镜像构建与推送 | ✅ | bubua12/redis-operator:v0.0.1 |
| 集群部署验证 | ✅ | 3 节点 Redis 集群正常运行 |

---

## 近期规划（v0.1.0）— 1~2 周

### 1. Finalizer 清理机制

**产品价值**：用户删除 Redis 资源时，自动清理关联的所有子资源（StatefulSet、Service、ConfigMap、PVC），避免资源泄露。

**用户场景**：
```yaml
# 用户执行 kubectl delete redis redis-sample
# 期望：所有关联资源（Pod、Service、PVC）全部清理干净
# 现状：StatefulSet 和 Service 可能残留（取决于 OwnerReference 是否生效）
```

**技术实现**：
- 在 Redis 对象上添加 Finalizer（`cache.bubua12.com/finalizer`）
- Reconcile 检测到删除事件时，先清理子资源，再移除 Finalizer
- 确保 PVC 数据不会意外丢失（可配置保留策略）

**验收标准**：
- [ ] 删除 Redis CR 后，StatefulSet、Service、PVC 自动清理
- [ ] 清理过程中 Operator 日志无报错
- [ ] 清理完成后 Redis CR 才被真正删除

---

### 2. 配置文件管理（ConfigMap）

**产品价值**：让用户通过 Spec 自定义 redis.conf 配置，无需挂载外部文件。

**用户场景**：
```yaml
apiVersion: cache.bubua12.com/v1alpha1
kind: Redis
metadata:
  name: redis-sample
spec:
  replicas: 3
  image: redis:7.2-alpine
  port: 6379
  config:
    maxmemory: "256mb"
    maxmemory-policy: "allkeys-lru"
    appendonly: "yes"
    appendfsync: "everysec"
```

**技术实现**：
- 新增 `RedisSpec.Config` 字段（`map[string]string`）
- Reconcile 自动创建 ConfigMap 挂载到 Pod
- 配置变更时自动重启 Pod（通过 StatefulSet annotation 触发）

**验收标准**：
- [ ] 用户指定的配置正确写入 redis.conf
- [ ] 配置变更后 Pod 自动滚动更新
- [ ] 未指定配置时使用合理的默认值

---

### 3. 密码管理（Secret）

**产品价值**：支持 Redis 密码认证，保障生产环境安全。

**用户场景**：
```yaml
apiVersion: cache.bubua12.com/v1alpha1
kind: Redis
metadata:
  name: redis-sample
spec:
  replicas: 3
  passwordSecretRef:
    name: my-redis-password
    key: password
```

**技术实现**：
- 新增 `RedisSpec.PasswordSecretRef` 字段
- 引用已有的 Secret，或自动生成随机密码
- 通过环境变量 `REDIS_PASSWORD` 注入容器
- redis.conf 自动添加 `requirepass` 配置

**验收标准**：
- [ ] 支持引用已有 Secret
- [ ] 支持自动生成密码并创建 Secret
- [ ] 密码变更后 Pod 自动重启生效
- [ ] 无密码时 Redis 不启用认证

---

### 4. 健康检查与自动恢复

**产品价值**：自动检测 Redis 实例健康状态，故障时自动重启。

**用户场景**：Redis Pod 内存溢出或进程崩溃时，Operator 自动检测并重启。

**技术实现**：
- 添加 `livenessProbe`（`redis-cli ping`）
- 添加 `readinessProbe`（`redis-cli ping`）
- StatefulSet 配置 `restartPolicy: Always`

**验收标准**：
- [ ] Redis 进程崩溃后自动重启
- [ ] 重启期间 Service 自动摘除不健康 Pod
- [ ] 恢复后自动加回 Service

---

## 中期规划（v0.2.0）— 1~2 月

### 5. 水平扩缩容

**产品价值**：用户修改 `replicas` 字段即可在线扩缩容，无需手动操作。

**用户场景**：
```yaml
# 从 3 个节点扩容到 5 个
spec:
  replicas: 5
```

**技术实现**：
- 直接更新 StatefulSet.Spec.Replicas
- 扩容时新 Pod 自动加入集群
- 缩容时优雅关闭（`SHUTDOWN NOSAVE` 或 `CLUSTER FORGET`）

**验收标准**：
- [ ] 扩容后新 Pod 正常加入集群
- [ ] 缩容时数据不丢失（主节点需先迁移）
- [ ] Status.ReadyReplicas 实时更新

---

### 6. 滚动更新

**产品价值**：升级 Redis 版本不中断服务。

**用户场景**：
```yaml
# 从 7.2 升级到 7.4
spec:
  image: redis:7.4-alpine
```

**技术实现**：
- 更新 StatefulSet 的 PodTemplate
- 使用 `RollingUpdate` 策略，逐个 Pod 更新
- 更新前等待当前 Pod 完成数据同步

**验收标准**：
- [ ] 镜像变更后自动触发滚动更新
- [ ] 更新过程中 Service 不中断
- [ ] Status.State 显示为 "Updating"

---

### 7. Webhook 参数校验

**产品价值**：用户提交无效配置时，立即返回清晰的错误提示，而不是等到运行时才发现。

**用户场景**：
```yaml
# 用户写了 replicas: 0，应该被拒绝
spec:
  replicas: 0
# 返回错误：replicas 最小值为 1
```

**技术实现**：
- ValidatingWebhook：校验 Spec 字段的合法性
- MutatingWebhook：设置默认值（image、port 等）

**验收标准**：
- [ ] replicas < 1 时拒绝创建
- [ ] image 为空时自动填充默认值
- [ ] 错误信息清晰可读

---

### 8. 持久化存储（PVC）

**产品价值**：数据持久化，Pod 重启不丢数据。

**用户场景**：
```yaml
spec:
  storage:
    size: 10Gi
    storageClassName: "standard"
```

**技术实现**：
- 新增 `RedisSpec.Storage` 字段
- StatefulSet 添加 `volumeClaimTemplates`
- 数据目录挂载到 `/data`

**验收标准**：
- [ ] PVC 自动创建并绑定到 Pod
- [ ] Pod 重启后数据不丢失
- [ ] 存储大小可配置

---

## 远期规划（v1.0.0）— 3~6 月

### 9. Redis Cluster 模式

**产品价值**：支持原生 Redis Cluster，实现数据分片和高可用。

**用户场景**：
```yaml
spec:
  mode: cluster
  cluster:
    masterCount: 3
    replicasPerMaster: 1
```

**技术实现**：
- 区分 Standalone / Sentinel / Cluster 三种模式
- Cluster 模式下自动执行 `CLUSTER MEET` 和 `CLUSTER REPLICATE`
- 使用 Headless Service 实现稳定的 DNS 发现

**验收标准**：
- [ ] 自动创建 6 节点 Cluster（3 主 3 从）
- [ ] 数据自动分片到 16384 个槽位
- [ ] 主节点故障时从节点自动提升

---

### 10. Sentinel 高可用

**产品价值**：自动故障转移，主节点挂了自动提升从节点。

**用户场景**：
```yaml
spec:
  mode: sentinel
  sentinel:
    replicas: 3
```

**技术实现**：
- 额外部署 Sentinel 进程（sidecar 或独立 Deployment）
- 监控主节点健康状态
- 故障时自动执行 `SENTINEL failover`

**验收标准**：
- [ ] 主节点故障 30 秒内自动切换
- [ ] 切换期间 Service 自动更新指向新主节点
- [ ] 数据不丢失

---

### 11. 监控与指标导出

**产品价值**：集成 Prometheus，实时监控 Redis 性能指标。

**用户场景**：
```yaml
spec:
  metrics:
    enabled: true
    image: oliver006/redis_exporter:latest
```

**技术实现**：
- Sidecar 容器运行 redis_exporter
- 暴露 `/metrics` 端点（9121）
- 自动创建 ServiceMonitor（Prometheus Operator）

**监控指标**：
- 连接数、QPS、内存使用、命中率
- 主从同步延迟、持久化状态

**验收标准**：
- [ ] Prometheus 可采集 Redis 指标
- [ ] Grafana Dashboard 模板可用
- [ ] 告警规则示例（内存 > 80%、连接数 > 1000）

---

### 12. 备份与恢复

**产品价值**：定期备份数据，支持快速恢复。

**用户场景**：
```yaml
spec:
  backup:
    enabled: true
    schedule: "0 2 * * *"  # 每天凌晨 2 点
    storage: "s3://my-bucket/redis-backup/"
```

**技术实现**：
- CronJob 定期执行 `BGSAVE`
- 上传 RDB 文件到 S3/MinIO
- 支持指定时间点恢复

**验收标准**：
- [ ] 按计划自动备份
- [ ] 备份文件可从 S3 下载
- [ ] 支持一键恢复到指定备份点

---

### 13. 资源限制与调度

**产品价值**：精确控制 CPU/内存使用，优化集群资源分配。

**用户场景**：
```yaml
spec:
  resources:
    requests:
      cpu: "100m"
      memory: "128Mi"
    limits:
      cpu: "500m"
      memory: "512Mi"
  nodeSelector:
    kubernetes.io/zone: "cn-east-1"
  tolerations:
    - key: "dedicated"
      operator: "Equal"
      value: "redis"
      effect: "NoSchedule"
```

**验收标准**：
- [ ] 资源限制正确应用到 Pod
- [ ] 支持 nodeSelector 节点选择
- [ ] 支持 tolerations 污点容忍
- [ ] 超出 limits 时 Pod 被 OOMKilled 并自动重启

---

### 14. 多集群联邦

**产品价值**：跨可用区部署，实现异地灾备。

**用户场景**：
```yaml
spec:
  topology:
    zones:
      - name: "cn-east-1"
        replicas: 2
      - name: "cn-east-2"
        replicas: 1
```

**技术实现**：
- 使用 PodTopologySpreadConstraints 分散部署
- 不同区的 Pod 使用不同的 StorageClass
- 跨区同步使用 Redis Replication

**验收标准**：
- [ ] Pod 自动分散到不同可用区
- [ ] 单区故障不影响其他区服务
- [ ] 故障恢复后自动同步数据

---

## 技术债务与优化

| 编号 | 项目 | 优先级 | 说明 |
|------|------|--------|------|
| T1 | 单元测试覆盖 | 高 | 当前测试覆盖率为 0，需要补充 envtest 测试 |
| T2 | E2E 测试 | 中 | 使用 Kind 集群进行端到端测试 |
| T3 | 日志规范化 | 中 | 统一日志格式，添加 traceID |
| T4 | Metrics 暴露 | 中 | Operator 自身的指标（Reconcile 次数、延迟） |
| T5 | 文档完善 | 中 | API 文档、部署文档、开发文档 |
| T6 | CI/CD 流水线 | 低 | GitHub Actions 自动构建、测试、发布 |
| T7 | 性能优化 | 低 | Reconcile 循环优化，减少 API 调用 |

---

## 版本发布计划

| 版本 | 目标日期 | 核心功能 |
|------|---------|---------|
| v0.0.1 | ✅ 2026-06-23 | 基础 Operator（CRD + Controller + 部署） |
| v0.1.0 | 2026-07-07 | Finalizer + ConfigMap + Secret + 健康检查 |
| v0.2.0 | 2026-08-04 | 扩缩容 + 滚动更新 + Webhook + PVC |
| v1.0.0 | 2026-12-01 | Cluster 模式 + Sentinel + 监控 + 备份 |

---

## 竞品对比

| 特性 | 我们的 Operator | Redis Operator (spotahome) | Redis Operator (OT-CONTAINER-KIT) |
|------|----------------|---------------------------|-----------------------------------|
| CRD 设计 | ✅ 简洁 | ⚠️ 复杂 | ✅ 简洁 |
| Cluster 模式 | ⬜ 规划中 | ✅ 支持 | ✅ 支持 |
| Sentinel | ⬜ 规划中 | ✅ 支持 | ✅ 支持 |
| 监控集成 | ⬜ 规划中 | ⚠️ 手动 | ✅ 自动 |
| 学习友好度 | ✅⭐⭐⭐⭐⭐ | ⚠️ | ⚠️ |
| 社区活跃度 | 🆕 新项目 | ⭐⭐⭐ | ⭐⭐⭐⭐ |

**我们的优势**：代码清晰、学习友好、适合入门 Kubernetes Operator 开发。

---

## 参与贡献

欢迎提交 Issue 和 Pull Request！

### 开发环境要求
- Go 1.25+
- Docker
- Kubernetes 集群（Kind / Minikube / 远程集群）
- kubebuilder v4+

### 快速开始
```bash
git clone https://github.com/bubua12/bubua12-redis-operator.git
cd bubua12-redis-operator
make generate && make manifests
make run  # 本地运行
```

---

> 💡 这份路线图会随着项目进展持续更新。如果你有功能建议或发现了 Bug，
> 欢迎提 Issue 讨论！
