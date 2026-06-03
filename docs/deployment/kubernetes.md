# Kubernetes 迁移兼容

## 原则

迁移到 Kubernetes 后，服务发现优先使用 K8s 原生 Service / DNS。应用代码保持不变，只调整配置和部署文件。

原因：

- K8s 已经根据 Pod 生命周期、readinessProbe、Service selector 和 EndpointSlice 自动维护服务发现。
- 应用不需要主动注册或注销实例。
- Nacos 可以继续作为配置中心使用，服务发现逐步切到 K8s DNS。

## 服务端配置

```yaml
grpc:
  addr: ":9000"

registry:
  enabled: false
```

K8s 下不需要应用主动注册服务实例。Pod 是否可被发现由 Service selector 和 readinessProbe 决定。

## Headless Service

推荐 gRPC 内部服务使用 Headless Service：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: goods-service
  namespace: livebid
spec:
  clusterIP: None
  selector:
    app: goods-service
  ports:
    - name: grpc
      port: 9000
      targetPort: 9000
```

Deployment 中配置 gRPC probe：

```yaml
readinessProbe:
  grpc:
    port: 9000
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 2
  failureThreshold: 3
livenessProbe:
  grpc:
    port: 9000
  initialDelaySeconds: 15
  periodSeconds: 10
  timeoutSeconds: 2
  failureThreshold: 3
```

调用方：

```yaml
goodsService:
  target: "dns:///goods-service.livebid.svc.cluster.local:9000"
```

readinessProbe 未通过的 Pod 不会进入 Service 对应的 EndpointSlice；这就是 K8s 阶段的实例健康摘除机制。

K8s 中 `api-gateway`、`ws-gateway` 的对外流量由 Ingress、LoadBalancer 或 NodePort 承接，业务 gRPC 服务只暴露集群内 Service。
