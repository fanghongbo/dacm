package g

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/fanghongbo/dacm/utils"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
)

var (
	cfg            = flag.String("c", "./config/cfg.json", "specify config file")
	v              = flag.Bool("v", false, "show version")
	vv             = flag.Bool("vv", false, "show version detail")
	ConfigFile     string
	configFileLock = new(sync.RWMutex)
	config         *GlobalConfig
)

type NacosNode struct {
	Ip   string `json:"ip"`
	Port int64  `json:"port"`
}

type Namespace struct {
	Id       string         `json:"id"`
	Name     string         `json:"name"`
	Username string         `json:"username"`
	Password string         `json:"password"`
	Configs  []*NacosConfig `json:"configs"`
}

type NacosConfig struct {
	DataId              string `json:"data_id"`
	Group               string `json:"group"`
	SyncFile            string `json:"sync_file"`
	Execute             string `json:"execute"`
	ExecuteDelay        int64  `json:"execute_delay"`
	ExecuteTimeout      int64  `json:"execute_timeout"`
	NotLoadCacheAtStart bool   `json:"not_load_cache_at_start"`
	Timeout             int64  `json:"timeout"`
}

type GlobalConfig struct {
	ClusterType  string       `json:"cluster_type"` // nacos | acm
	ClusterNodes []*NacosNode `json:"cluster_nodes"`
	Endpoint     string       `json:"endpoint"`
	RegionId     string       `json:"region_id"`
	AccessKey    string       `json:"access_key"`
	SecretKey    string       `json:"secret_key"`
	OpenKms      bool         `json:"open_kms"`
	Namespaces   []*Namespace `json:"namespaces"`
	MaxCPURate   float64      `json:"max_cpu_rate"`
	MaxMemRate   float64      `json:"max_mem_rate"`
	CacheDir     string       `json:"cache_dir"`
	LogDir       string       `json:"log_dir"`
	LogLevel     string       `json:"log_level"`
	RotateTime   string       `json:"rotate_time"`
	MaxAge       int64        `json:"max_age"`
}

func Conf() *GlobalConfig {
	configFileLock.RLock()
	defer configFileLock.RUnlock()

	return config
}

func InitConfig() error {
	var (
		cfgFile   string
		bs        []byte
		err       error
		maxMemMB  int
		maxCPUNum int
	)

	flag.Parse()

	if *v {
		fmt.Println(VersionInfo())
		os.Exit(0)
	}

	if *vv {
		fmt.Println(VersionDetailInfo())
		os.Exit(0)
	}

	cfgFile = *cfg
	ConfigFile = cfgFile

	if cfgFile == "" {
		return errors.New("config file not specified: use -c $filename")
	}

	if _, err = os.Stat(cfgFile); os.IsNotExist(err) {
		return fmt.Errorf("config file specified not found: %s", cfgFile)
	} else {
		log.Printf("[INFO] use config file: %s", ConfigFile)
	}

	if bs, err = ioutil.ReadFile(cfgFile); err != nil {
		return fmt.Errorf("read config file failed: %s", err.Error())
	} else {
		if err = json.Unmarshal(bs, &config); err != nil {
			return fmt.Errorf("decode config file failed: %s", err.Error())
		} else {
			log.Printf("[INFO] load config success from %s", cfgFile)
		}
	}

	if err = Validator(); err != nil {
		return fmt.Errorf("validator config file fail: %s", err)
	}

	// 最大使用内存
	maxMemMB = utils.CalculateMemLimit(config.MaxMemRate)

	// 最大cpu核数
	maxCPUNum = utils.GetCPULimitNum(config.MaxCPURate)

	log.Printf("[INFO] bind [%d] cpu core", maxCPUNum)
	runtime.GOMAXPROCS(maxCPUNum)

	log.Printf("[INFO] memory limit: %d MB", maxMemMB)

	return nil
}

func ReloadConfig() error {
	var (
		bs  []byte
		err error
	)

	if _, err = os.Stat(ConfigFile); os.IsNotExist(err) {
		return fmt.Errorf("config file specified not found: %s", ConfigFile)
	} else {
		log.Printf("[INFO] reload config file: %s", ConfigFile)
	}

	if bs, err = ioutil.ReadFile(ConfigFile); err != nil {
		return fmt.Errorf("reload config file failed: %s", err)
	} else {
		configFileLock.RLock()
		defer configFileLock.RUnlock()

		if err = json.Unmarshal(bs, &config); err != nil {
			return fmt.Errorf("decode config file failed: %s", err)
		} else {
			log.Printf("[INFO] reload config success from %s", ConfigFile)
		}
	}

	if err = Validator(); err != nil {
		return fmt.Errorf("validator config file fail: %s", err)
	}

	return nil
}

func Validator() error {

	if !utils.InStringArray([]string{"acm", "nacos"}, config.ClusterType) {
		return fmt.Errorf("cluster type not support: %s", config.ClusterType)
	}

	if config.RotateTime == "" {
		config.RotateTime = "1h"
	}

	if config.MaxAge <= 0 {
		config.MaxAge = 3
	}

	if config.LogLevel == "" {
		config.LogLevel = "info"
	} else {
		if !utils.InStringArray([]string{"info", "warn", "error", "debug"}, config.LogLevel) {
			return fmt.Errorf("log level must be debug, info, warn, error")
		}
	}

	if config.ClusterType == "acm" {
		if config.Endpoint == "" {
			return fmt.Errorf("acm service endpoint is empty")
		}

		if config.RegionId == "" {
			return fmt.Errorf("acm service region id is empty")
		}

		if config.AccessKey == "" {
			return fmt.Errorf("acm service access key is empty")
		}

		if config.SecretKey == "" {
			return fmt.Errorf("acm service secret key is empty")
		}
	} else {
		var exist bool
		for _, server := range config.ClusterNodes {
			if server.Ip == "" {
				return fmt.Errorf("nacos server host is empty")
			}

			if server.Port <= 0 {
				return fmt.Errorf("nacos server port is empty")
			}

			exist = true
		}

		if !exist {
			return fmt.Errorf("nacos cluster nodes is empty")
		}
	}

	for _, namespace := range config.Namespaces {
		if namespace.Id == "" {
			return fmt.Errorf("namespace id is empty")
		}

		if namespace.Name == "" {
			return fmt.Errorf("namespace name is empty")
		}

		for _, item := range namespace.Configs {
			if item.DataId != "" {
				if item.Group == "" {
					item.Group = "DEFAULT_GROUP"
				}

				if item.ExecuteDelay < 0 {
					return fmt.Errorf("execute delay must be ge 0")
				}

				if item.ExecuteTimeout <= 0 {
					return fmt.Errorf("execute timeout must be gt 0")
				}

				if item.Timeout <= 0 {
					return fmt.Errorf("listen config timeout must be gt 0")
				}

				if item.ExecuteDelay < 0 {
					return fmt.Errorf("get nacos config timeout must be ge 0")
				}
			}
		}
	}

	// MaxCPURate
	if config.MaxCPURate < 0 || config.MaxCPURate > 1 {
		return errors.New("max_cpu_rate is range 0 to 1")
	}

	// MaxMemRate
	if config.MaxMemRate < 0 || config.MaxMemRate > 1 {
		return errors.New("max_mem_rate is range 0 to 1")
	}

	return nil
}

func ReformatConfigValue(value string) string {
	return strings.TrimSpace(value)
}
