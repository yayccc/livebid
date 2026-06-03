package nacosx

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type Registration struct {
	client naming_client.INamingClient
	param  vo.RegisterInstanceParam
	log    *zap.Logger

	cancel context.CancelFunc
	once   sync.Once
}

// 注册实例到注册中心，并返回一个Registration对象用于后续的健康检查和注销
func RegisterInstance(ctx context.Context, client naming_client.INamingClient, cfg RegistryConfig, listenAddr net.Addr, env string, log *zap.Logger) (*Registration, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if strings.ToLower(cfg.Provider) != "nacos" {
		return nil, fmt.Errorf("unsupported registry provider %q", cfg.Provider)
	}
	if client == nil {
		return nil, fmt.Errorf("nacos naming client is required when registry is enabled")
	}
	NormalizeRegistry(&cfg, cfg.ServiceName)
	ip := cfg.IP
	if ip == "" {
		ip = detectRegisterIP(listenAddr)
	}
	port := cfg.Port
	if port == 0 {
		parsedPort, err := portFromAddr(listenAddr)
		if err != nil {
			return nil, err
		}
		port = parsedPort
	}
	metadata := map[string]string{
		"protocol": "grpc",
		"env":      env,
	}
	for key, value := range cfg.Metadata {
		metadata[key] = value
	}
	param := vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        port,
		Weight:      cfg.Weight,
		Enable:      true,
		Healthy:     true,
		Metadata:    metadata,
		ClusterName: cfg.Cluster,
		ServiceName: cfg.ServiceName,
		GroupName:   cfg.Group,
		Ephemeral:   true,
	}
	ok, err := client.RegisterInstance(param)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("nacos register instance returned false")
	}
	if log != nil {
		log.Info("registered grpc instance to nacos",
			zap.String("service", cfg.ServiceName),
			zap.String("group", cfg.Group),
			zap.String("cluster", cfg.Cluster),
			zap.String("ip", ip),
			zap.Uint64("port", port),
		)
	}
	_ = ctx
	return &Registration{client: client, param: param, log: log}, nil
}

func (r *Registration) StartHealthCheck(ctx context.Context, cfg HealthCheckConfig) {
	if r == nil || !cfg.Enabled {
		return
	}
	NormalizeHealthCheck(&cfg)
	checkCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	go r.healthLoop(checkCtx, cfg)
}

func (r *Registration) Deregister(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var err error
	r.once.Do(func() {
		if r.cancel != nil {
			r.cancel()
		}
		_, _ = r.updateHealthy(false)
		_, err = r.client.DeregisterInstance(vo.DeregisterInstanceParam{
			Ip:          r.param.Ip,
			Port:        r.param.Port,
			Cluster:     r.param.ClusterName,
			ServiceName: r.param.ServiceName,
			GroupName:   r.param.GroupName,
			Ephemeral:   r.param.Ephemeral,
		})
		if r.log != nil {
			r.log.Info("deregistered grpc instance from nacos",
				zap.String("service", r.param.ServiceName),
				zap.String("ip", r.param.Ip),
				zap.Uint64("port", r.param.Port),
			)
		}
	})
	_ = ctx
	return err
}

func (r *Registration) healthLoop(ctx context.Context, cfg HealthCheckConfig) {
	target := net.JoinHostPort(r.param.Ip, strconv.FormatUint(r.param.Port, 10))
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		if r.log != nil {
			r.log.Warn("create grpc health client failed", zap.String("target", target), zap.Error(err))
		}
		return
	}
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)
	ticker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	failures := 0
	healthy := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
			resp, err := client.Check(checkCtx, &healthpb.HealthCheckRequest{})
			cancel()
			ok := err == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
			if ok {
				failures = 0
				if !healthy {
					if updated, updateErr := r.updateHealthy(true); updateErr == nil && updated {
						healthy = true
					}
				}
				continue
			}
			failures++
			if failures >= cfg.FailureThreshold && healthy {
				if updated, updateErr := r.updateHealthy(false); updateErr == nil && updated {
					healthy = false
				}
			}
		}
	}
}

func (r *Registration) updateHealthy(healthy bool) (bool, error) {
	param := vo.UpdateInstanceParam{
		Ip:          r.param.Ip,
		Port:        r.param.Port,
		Weight:      r.param.Weight,
		Enable:      r.param.Enable,
		Healthy:     healthy,
		Metadata:    r.param.Metadata,
		ClusterName: r.param.ClusterName,
		ServiceName: r.param.ServiceName,
		GroupName:   r.param.GroupName,
		Ephemeral:   r.param.Ephemeral,
	}
	ok, err := r.client.UpdateInstance(param)
	if err != nil && r.log != nil {
		r.log.Warn("update nacos instance health failed",
			zap.String("service", r.param.ServiceName),
			zap.Bool("healthy", healthy),
			zap.Error(err),
		)
	}
	return ok, err
}

func portFromAddr(addr net.Addr) (uint64, error) {
	tcpAddr, ok := addr.(*net.TCPAddr)
	if ok {
		return uint64(tcpAddr.Port), nil
	}
	_, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.ParseUint(port, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func detectRegisterIP(addr net.Addr) string {
	if tcpAddr, ok := addr.(*net.TCPAddr); ok && tcpAddr.IP != nil && !tcpAddr.IP.IsUnspecified() {
		return tcpAddr.IP.String()
	}
	if ip := firstNonLoopbackIPv4(); ip != "" {
		return ip
	}
	return "127.0.0.1"
}

func firstNonLoopbackIPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		var ip net.IP
		switch value := addr.(type) {
		case *net.IPNet:
			ip = value.IP
		case *net.IPAddr:
			ip = value.IP
		}
		if ip == nil || ip.IsLoopback() {
			continue
		}
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return ""
}
