package vo

import "github.com/fanghongbo/dacm/common/pkg/nacos-sdk-go/common/constant"

type NacosClientParam struct {
	ClientConfig  *constant.ClientConfig  // optional
	ServerConfigs []constant.ServerConfig // optional
}
