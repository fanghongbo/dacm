package g

import (
	"errors"
	"github.com/fanghongbo/dacm/utils"
	"github.com/fanghongbo/dacm/common/pkg/nacos-sdk-go/common/logger"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	Pwd     string
	LocalIp string
)

func InitRuntime() error {
	var err error

	// 内存监控
	go MemMonitor()

	// 当前主机默认的ip地址
	if LocalIp, err = getLocalDefaultIp(); err != nil {
		return err
	}

	// 当前程序运行路径
	if Pwd, err = GetPwd(); err != nil {
		return err
	}

	return nil
}

func MemMonitor() {
	var (
		nowMemUsedMB uint64
		maxMemMB     uint64
		rate         uint64
	)

	for {
		time.Sleep(time.Second * 10)

		nowMemUsedMB = getMemUsedMB()
		maxMemMB = uint64(utils.CalculateMemLimit(config.MaxMemRate))
		rate = (nowMemUsedMB * 100) / maxMemMB

		// 若超50%限制，打印 warning
		if rate > 50 {
			logger.Warn("[dacm] heap memory used rate, current: %d%%", rate)
		}

		// 超过100%，就退出了
		if rate > 100 {
			// 堆内存已超过限制，退出进程
			logger.Errorf("[dacm] heap memory size over limit. quit process.[used:%dMB][limit:%dMB][rate:%d]", nowMemUsedMB, maxMemMB, rate)
			os.Exit(127)
		}
	}
}

func getMemUsedMB() uint64 {
	var (
		sts runtime.MemStats
		ret uint64
	)

	runtime.ReadMemStats(&sts)
	// 这里取了mem.Alloc
	ret = sts.HeapAlloc / 1024 / 1024
	return ret
}

// 获取程序运行的目录
func GetPwd() (string, error) {
	var (
		pwd string
		err error
	)

	if pwd, err = os.Executable(); err != nil {
		return "", err
	}

	return filepath.Dir(pwd), nil
}

func getLocalDefaultIp() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iterm := range interfaces {
		if iterm.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iterm.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}

		address, err := iterm.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range address {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip.String(), nil
		}
	}

	return "", errors.New("找不到本机默认ip")
}

func getIpFromAddr(addr net.Addr) net.IP {
	var ip net.IP

	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}

	if ip == nil || ip.IsLoopback() {
		return nil
	}

	ip = ip.To4()
	if ip == nil {
		return nil // not an ipv4 address
	}

	return ip
}
