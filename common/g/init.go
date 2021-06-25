package g

import (
	"context"
)

func InitAll() error {
	var err error

	// 初始化运行时环境
	if err = InitRuntime(); err != nil {
		return err
	}

	// 初始化配置文件
	if err = InitConfig(); err != nil {
		return err
	}

	return nil
}

func Shutdown(ctx context.Context) error {
	var (
		ch  chan struct{}
	)

	ch = make(chan struct{}, 0)

	go func() {
		ch <- struct{}{}
	}()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return nil
	}
}