package task

import (
	"context"
	"fmt"
	"github.com/fanghongbo/dacm/common/g"
	"github.com/fanghongbo/dacm/common/pkg/nacos-sdk-go/clients"
	"github.com/fanghongbo/dacm/common/pkg/nacos-sdk-go/clients/config_client"
	"github.com/fanghongbo/dacm/common/pkg/nacos-sdk-go/common/constant"
	"github.com/fanghongbo/dacm/common/pkg/nacos-sdk-go/common/logger"
	"github.com/fanghongbo/dacm/common/pkg/nacos-sdk-go/vo"
	"github.com/fanghongbo/dacm/utils"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

var lock sync.Mutex

func init() {
	lock = sync.Mutex{} // 全局文件写锁
}

func Start() error {
	var (
		globalConfig *g.GlobalConfig
	)

	globalConfig = g.Conf()

	log.Printf("[INFO] logDir:<%s>   cacheDir:<%s>", globalConfig.LogDir, globalConfig.CacheDir)

	for _, namespace := range globalConfig.Namespaces {
		for _, config := range namespace.Configs {
			var (
				configClient config_client.IConfigClient
				content      string
				err          error
			)

			log.Printf("[INFO] init config: <%s@@%s@@%s@@%s>", config.DataId, config.Group, namespace.Name, namespace.Id)

			if globalConfig.ClusterType == "acm" {
				if configClient, err = getAcmClient(globalConfig, namespace, config); err != nil {
					return err
				}
			} else if globalConfig.ClusterType == "nacos" {
				if configClient, err = getNacosClient(globalConfig, namespace, config); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unkonw cluster type: %s", globalConfig.ClusterType)
			}

			// 获取指定配置文件的内容
			if content, err = configClient.GetConfig(vo.ConfigParam{
				DataId: config.DataId,
				Group:  config.Group,
			}); err != nil {
				return fmt.Errorf("get namespace: %s group: %s dataId: %s err: %s", namespace.Name, config.Group, config.DataId, err.Error())
			}

			logger.Debugf("[client.GetConfig] namespace: <%s> group: <%s> dataId: <%s> content: \n%s", namespace.Name, config.Group, config.DataId, content)
			if err = updateFile(config.SyncFile, config.Execute, config.ExecuteDelay, config.ExecuteTimeout, content); err != nil {
				return err
			}

			// 创建监听任务
			go func(config *g.NacosConfig) {
				if err = configClient.ListenConfig(vo.ConfigParam{
					DataId: config.DataId,
					Group:  config.Group,
					OnChange: func(namespace, group, dataId, content string) {
						if err = updateFile(config.SyncFile, config.Execute, config.ExecuteDelay, config.ExecuteTimeout, content); err != nil {
							logger.Errorf("[client.ListenConfig] %s", err.Error())
						}
					},
				}); err != nil {
					logger.Errorf("[client.ListenConfig] %s", err.Error())
				}
			}(config)
		}
	}

	return nil
}

func getNacosClient(globalConfig *g.GlobalConfig, namespace *g.Namespace, config *g.NacosConfig) (config_client.IConfigClient, error) {
	var (
		serverConfig []constant.ServerConfig
		clientConfig constant.ClientConfig
		configClient config_client.IConfigClient
		err          error
	)

	serverConfig = []constant.ServerConfig{}
	for _, node := range globalConfig.ClusterNodes {
		serverConfig = append(serverConfig, constant.ServerConfig{
			IpAddr: node.Ip,
			Port:   uint64(node.Port),
		})
	}

	clientConfig = constant.ClientConfig{
		Username:            namespace.Username,
		Password:            namespace.Password,
		NamespaceId:         namespace.Id,
		TimeoutMs:           uint64(config.Timeout),
		NotLoadCacheAtStart: config.NotLoadCacheAtStart,
		LogDir:              globalConfig.LogDir,
		CacheDir:            globalConfig.CacheDir,
		RotateTime:          globalConfig.RotateTime,
		MaxAge:              globalConfig.MaxAge,
		LogLevel:            globalConfig.LogLevel,
	}

	if configClient, err = clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfig,
		},
	); err != nil {
		return nil, err
	}

	return configClient, nil
}

func getAcmClient(globalConfig *g.GlobalConfig, namespace *g.Namespace, config *g.NacosConfig) (config_client.IConfigClient, error) {
	var (
		clientConfig constant.ClientConfig
		configClient config_client.IConfigClient
		err          error
	)

	clientConfig = constant.ClientConfig{
		Endpoint:            globalConfig.Endpoint,
		RegionId:            globalConfig.RegionId,
		AccessKey:           globalConfig.AccessKey,
		SecretKey:           globalConfig.SecretKey,
		Username:            namespace.Username,
		Password:            namespace.Password,
		NamespaceId:         namespace.Id,
		TimeoutMs:           uint64(config.Timeout),
		NotLoadCacheAtStart: config.NotLoadCacheAtStart,
		LogDir:              globalConfig.LogDir,
		CacheDir:            globalConfig.CacheDir,
		RotateTime:          globalConfig.RotateTime,
		MaxAge:              globalConfig.MaxAge,
		LogLevel:            globalConfig.LogLevel,
	}

	if configClient, err = clients.CreateConfigClient(map[string]interface{}{
		"clientConfig": clientConfig,
	}); err != nil {
		return nil, err
	}

	return configClient, nil
}

func checkFile(localFile string, content string) (bool, error) {
	var (
		exist            bool
		file             *os.File
		localFileContent []byte
		hasChange        bool
		err              error
	)

	lock.Lock()
	defer lock.Unlock()

	// 检查同步文件是否存在, 如果存在则检查配置是否有变更, 如果有则更新
	if exist, err = utils.IsFileExist(localFile); err != nil {
		return hasChange, err
	}

	if exist {
		// 检查本地文件和配置中心对比是否发生了变化
		if localFileContent, err = utils.ReadFile(localFile); err != nil {
			return false, err
		}

		if string(localFileContent) != content {
			// 配置变更, 更新本地配置文件
			if file, err = os.OpenFile(localFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0766); err != nil {
				return hasChange, err
			}

			// 标记文件已经发生了变化
			hasChange = true
			logger.Debugf("[client.UpdateFile] file: <%s> has change, new content: \n%s", localFile, content)
		} else {
			logger.Debugf("[client.UpdateFile] file: <%s> no change", localFile)
		}

	} else {
		// 配置文件不存在，则创建并写入文件
		if file, err = os.Create(localFile); err != nil {
			return hasChange, err
		}

		// 标记文件已经发生了变化
		hasChange = true

		logger.Debugf("[client.UpdateFile] new file: <%s> content: \n%s", localFile, content)
	}

	if file != nil {
		defer func() {
			_ = file.Close()
		}()

		if _, err = file.WriteString(content); err != nil {
			return hasChange, err
		}
	}

	return hasChange, nil
}

func updateFile(localFile string, command string, delay int64, timeout int64, content string) error {
	var (
		hasChange   bool
		taskId      string
		commandName string
		commandArgs string
		ctx         context.Context
		done        chan interface{}
		e           chan error
		cmd         *exec.Cmd
		err         error
	)

	if hasChange, err = checkFile(localFile, content); err != nil {
		return err
	}

	if !hasChange {
		return nil
	}

	if command == "" {
		return nil
	}

	taskId = utils.GetUuid()

	// 延迟执行
	logger.Debugf("[client.execute] taskId: <%s> execute: <%s> after %d ms", taskId, command, delay)
	time.Sleep(time.Duration(delay) * time.Millisecond)

	switch runtime.GOOS {
	case "windows":
		commandName = "cmd"
		commandArgs = "/C"
	default:
		commandName = "bash"
		commandArgs = "-c"
	}

	// 执行更新操作命令
	ctx, _ = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)

	cmd = exec.Command(commandName, commandArgs, command)

	done = make(chan interface{})
	e = make(chan error, 0)

	go func(done chan interface{}, e chan error) {
		var (
			stdout io.ReadCloser
			stderr io.ReadCloser
			err    error
		)

		if stdout, err = cmd.StdoutPipe(); err != nil {
			e <- err
			return
		}

		if stderr, err = cmd.StderrPipe(); err != nil {
			e <- err
			return
		}

		// 实时打印执行日志
		go syncLog(taskId, stdout, "stdout")
		go syncLog(taskId, stderr, "stderr")

		if err = cmd.Start(); err != nil {
			e <- err
			return
		}

		if err = cmd.Wait(); err != nil {
			e <- err
			return
		}

		done <- 1
	}(done, e)

	select {
	case <-done:
		return nil
	case err := <-e:
		return err
	case <-ctx.Done():
		return fmt.Errorf("taskId: <%s> execute timeout", taskId)
	}
}

func syncLog(taskId string, reader io.ReadCloser, logType string) {
	var buf []byte

	buf = make([]byte, 1024, 1024)
	for {
		strNum, err := reader.Read(buf)
		if strNum > 0 {
			outputByte := buf[:strNum]
			logger.Infof("[task.%s] taskId: <%s> result: \n%s", logType, taskId, string(outputByte))
		}

		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "file already closed") {
				return
			} else {
				logger.Errorf("[task.execute.%s] taskId: <%s> err: %s", logType, taskId, err.Error())
				return
			}
		}
	}
}
