package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type MyRegistryDiscovery struct {
	*MultiServersDiscovery
	registry 	string	// 注册中心地址
	timeout		time.Duration	// 服务列表的过期时间，过期后需要重新获取
	lastUpdate 	time.Time	// 从注册中心更新服务列表的时间
}

const defaultUpdateTimeout = time.Second * 10

// 创建一个实例
func NewMyRegistryDiscovery(registerAddr string, timeout time.Duration) *MyRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	d := &MyRegistryDiscovery{
		MultiServersDiscovery: 	NewMultiServerDiscovery(make([]string, 0)),
		registry: 				registerAddr,
		timeout:				timeout,
	}
	return d
}

// 手动更新服务器列表
func (d *MyRegistryDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	d.lastUpdate = time.Now()
	return nil
}

// 从注册中心刷新服务器列表
func (d *MyRegistryDiscovery) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	// 若还未过期，直接返回
	if d.lastUpdate.Add(d.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc registry: refresh servers from registry", d.registry)
	// 从注册中心获取服务
	resp, err := http.Get(d.registry)
	if err != nil {
		log.Println("rpc registry refresh err:", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("X-Myrpc-Servers"), ",")
	d.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			d.servers = append(d.servers, server)
		}
	}
	// 更新时间
	d.lastUpdate = time.Now()
	return nil
}

// 获取服务
func (d *MyRegistryDiscovery) Get(mode SelectMode) (string, error) {
	if err := d.Refresh(); err != nil {
		return "", err
	}
	return d.MultiServersDiscovery.Get(mode)
}

// 获取所有服务
func (d *MyRegistryDiscovery) GetAll() ([]string, error) {
	if err := d.Refresh(); err != nil {
		return nil, err
	}
	return d.MultiServersDiscovery.GetAll()
}


