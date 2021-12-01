package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// 选择不同的负载均衡策略
type SelectMode int

const (
	RandomSelect     SelectMode = iota // select randomly
	RoundRobinSelect                   // select using Robbin algorithm
)

type Discovery interface {
	Refresh() error // refresh from remote registry
	Update(servers []string) error // 手动更新服务列表
	Get(mode SelectMode) (string, error) // 根据负载均衡策略，选择一个服务实例
	GetAll() ([]string, error) // 返回所有的服务实例
}

var _ Discovery = (*MultiServersDiscovery)(nil)

type MultiServersDiscovery struct {
	rd 		*rand.Rand // 随机数
	mu 		sync.RWMutex
	servers []string // 已注册的服务
	index 	int // 位置
}

func (d *MultiServersDiscovery) Refresh() error {
	return nil
}

// Update the servers of discovery dynamically if needed
func (d *MultiServersDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}

// Get 根据对应的模式选择一个服务
func (d *MultiServersDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return d.servers[d.rd.Intn(n)], nil
	case RoundRobinSelect:
		ser := d.servers[d.index % n]
		d.index = (d.index + 1) % n
		return ser, nil
	default:
		return "", errors.New("rpc discovery: not supported select mode")
	}
}

// GetAll 获取所有的服务
func (d *MultiServersDiscovery) GetAll() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	servers := make([]string, len(d.servers), len(d.servers))
	copy(servers, d.servers)
	return servers, nil
}

// NewMultiServerDiscovery 获取一个实例
func NewMultiServerDiscovery(servers []string) *MultiServersDiscovery {
	d := &MultiServersDiscovery{
		servers: servers,
		rd:	rand.New(rand.NewSource(time.Now().UnixNano())), // 产生随机数
	}
	d.index = d.rd.Intn(math.MaxInt32 - 1)
	return d
}