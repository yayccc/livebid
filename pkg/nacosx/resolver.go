package nacosx

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"google.golang.org/grpc/resolver"
)

const Scheme = "nacosx"

// globalResolver 在包初始化时注册到 gRPC，全进程共享同一个 Nacos resolver。
// 业务启动时通过 SetDefaultNamingClient 注入 Nacos naming client，之后
// grpc.NewClient("nacosx:///goods-service") 才能解析到具体实例地址。
var globalResolver = &resolverBuilder{
	group:   DefaultGroup,
	cluster: DefaultCluster,
}

func init() {
	resolver.Register(globalResolver)
}

func SetDefaultNamingClient(client naming_client.INamingClient) {
	globalResolver.mu.Lock()
	defer globalResolver.mu.Unlock()
	globalResolver.client = client
}

func SetResolverDefaults(group string, cluster string) {
	globalResolver.mu.Lock()
	defer globalResolver.mu.Unlock()
	if group != "" {
		globalResolver.group = group
	}
	if cluster != "" {
		globalResolver.cluster = cluster
	}
}

type resolverBuilder struct {
	mu      sync.RWMutex
	client  naming_client.INamingClient
	group   string
	cluster string
}

func (b *resolverBuilder) Scheme() string {
	return Scheme
}

func (b *resolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	b.mu.RLock()
	client := b.client
	group := b.group
	cluster := b.cluster
	b.mu.RUnlock()
	if client == nil {
		return nil, fmt.Errorf("nacos resolver requires a naming client")
	}
	if value := target.URL.Query().Get("group"); value != "" {
		group = value
	}
	if value := target.URL.Query().Get("cluster"); value != "" {
		cluster = value
	}
	// nacosx:///goods-service 的 Endpoint() 结果是 goods-service。
	// 也支持 nacosx:///goods-service?group=LIVEBID&cluster=default 覆盖默认分组。
	serviceName := target.Endpoint()
	if serviceName == "" {
		return nil, fmt.Errorf("nacos target requires service name")
	}
	r := &nacosResolver{
		client:      client,
		cc:          cc,
		serviceName: serviceName,
		group:       group,
		clusters:    []string{cluster},
	}
	r.callback = r.updateFromInstances
	if err := r.resolve(); err != nil {
		return nil, err
	}
	// 首次解析后继续订阅服务实例变化，Nacos 有实例上下线时会回调并刷新 gRPC 地址列表。
	_ = client.Subscribe(&vo.SubscribeParam{
		ServiceName:       serviceName,
		GroupName:         group,
		Clusters:          r.clusters,
		SubscribeCallback: r.callback,
	})
	return r, nil
}

type nacosResolver struct {
	client      naming_client.INamingClient
	cc          resolver.ClientConn
	serviceName string
	group       string
	clusters    []string
	callback    func([]model.Instance, error)
}

func (r *nacosResolver) ResolveNow(resolver.ResolveNowOptions) {
	// gRPC 在连接失败或需要刷新解析结果时会调用 ResolveNow，这里主动重新查询 Nacos。
	_ = r.resolve()
}

func (r *nacosResolver) Close() {
	// ClientConn 关闭时解除订阅，避免 resolver 生命周期结束后仍收到实例变更回调。
	_ = r.client.Unsubscribe(&vo.SubscribeParam{
		ServiceName:       r.serviceName,
		GroupName:         r.group,
		Clusters:          r.clusters,
		SubscribeCallback: r.callback,
	})
}

func (r *nacosResolver) resolve() error {
	// 只取健康实例；负载均衡交给 grpcx 里的 round_robin，而不是使用 Nacos 的单实例选择。
	instances, err := r.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: r.serviceName,
		GroupName:   r.group,
		Clusters:    r.clusters,
		HealthyOnly: true,
	})
	if err != nil {
		r.cc.ReportError(err)
		return err
	}
	r.updateFromInstances(instances, nil)
	return nil
}

func (r *nacosResolver) updateFromInstances(instances []model.Instance, err error) {
	if err != nil {
		r.cc.ReportError(err)
		return
	}
	addresses := make([]resolver.Address, 0, len(instances))
	for _, instance := range instances {
		// Nacos 返回的实例可能包含禁用、不健康或权重为 0 的记录，进入 gRPC 前再兜底过滤一次。
		if !instance.Healthy || !instance.Enable || instance.Weight <= 0 {
			continue
		}
		// 同一个 Nacos service 理论上可能注册多种协议，这里只把 gRPC 实例交给 gRPC client。
		if instance.Metadata != nil && instance.Metadata["protocol"] != "" && instance.Metadata["protocol"] != "grpc" {
			continue
		}
		addresses = append(addresses, resolver.Address{
			Addr:       net.JoinHostPort(instance.Ip, strconv.FormatUint(instance.Port, 10)),
			ServerName: strings.TrimPrefix(instance.ServiceName, "DEFAULT_GROUP@@"),
		})
	}
	// 会把最新地址列表推给 gRPC，后续由 round_robin 在这些 SubConn 之间轮询。
	_ = r.cc.UpdateState(resolver.State{Addresses: addresses})
}
