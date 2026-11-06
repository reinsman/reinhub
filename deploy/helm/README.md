# examplesrv Helm Chart

## 安装

默认值与 `deploy/kubernetes` 中的现有部署保持一致，包括镜像、NodePort 和 hostPath。安装前至少应覆盖 JWT key、数据库密码和数据库地址：

```bash
helm upgrade --install examplesrv ./deploy/helm \
  --namespace iam \
  --create-namespace \
  --set-string config.jwtKey='<jwt-key>' \
  --set-string config.database.password='<database-password>' \
  --set-string config.database.endpoint='<mysql-host>:3306'
```

生产环境推荐自行准备完整的 `conf.toml` Secret，避免通过 Helm values 保存敏感值：

```bash
kubectl create secret generic examplesrv-config \
  --namespace iam \
  --from-file=conf.toml=/path/to/conf.toml

helm upgrade --install examplesrv ./deploy/helm \
  --namespace iam \
  --create-namespace \
  --set-string config.existingSecret=examplesrv-config
```

## 持久化

Chart 默认保留现有清单中的 `hostPath`。若希望由 Helm 创建 PVC，请将 `persistence.hostPath` 设为空，并按集群情况覆盖 StorageClass：

```bash
helm upgrade --install examplesrv ./deploy/helm \
  --namespace iam \
  --set-string persistence.hostPath='' \
  --set-string persistence.storageClass=local-path
```

也可以通过 `persistence.existingClaim` 使用已有 PVC。

## 检查与卸载

```bash
helm lint ./deploy/helm
helm template examplesrv ./deploy/helm --namespace iam
helm uninstall examplesrv --namespace iam
```
