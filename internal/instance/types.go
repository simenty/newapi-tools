// Package instance provides multi-instance management for newapi-tools.
package instance

import "time"

// Instance represents a single newapi deployment instance.
type Instance struct {
	Name           string `json:"name"`            // 实例名（唯一标识）
	Home           string `json:"home"`            // 安装目录
	Port           int    `json:"port"`            // 服务端口
	DockerImage    string `json:"docker_image"`    // Docker 镜像
	Domain         string `json:"domain"`          // 域名
	HealthTimeout  int    `json:"health_timeout"`  // 健康检查超时时间
	MaxBackups     int    `json:"max_backups"`     // 最大保留备份数
	ComposeProject string `json:"compose_project"` // Compose 项目名
	CreatedAt      string `json:"created_at"`      // 创建时间（RFC3339）
	Active         bool   `json:"active"`          // 是否为当前活跃实例
}

// NewInstance creates a new Instance with default values and the current timestamp.
func NewInstance(name, home string, port int, dockerImage string) *Instance {
	return &Instance{
		Name:           name,
		Home:           home,
		Port:           port,
		DockerImage:    dockerImage,
		Domain:         "",
		HealthTimeout:  120,
		MaxBackups:     10,
		ComposeProject: "newapi-" + name,
		CreatedAt:      time.Now().Format(time.RFC3339),
		Active:         false,
	}
}
