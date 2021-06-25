package main

import (
	"context"
	"github.com/fanghongbo/dacm/common/g"
	"github.com/fanghongbo/dacm/common/pkg/nacos-sdk-go/common/logger"
	"github.com/fanghongbo/dacm/task"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var (
		quit   chan os.Signal
		ctx    context.Context
		cancel context.CancelFunc
		err    error
	)

	if err = g.InitAll(); err != nil {
		log.Fatalf("[ERROR] %s", err.Error())
	}

	if err = task.Start(); err != nil {
		log.Fatalf("[ERROR] %s", err.Error())
	}

	// 等待中断信号以优雅地关闭 Api（设置 5 秒的超时时间）
	quit = make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	<-quit
	logger.Warn("[dacm] Exiting ..")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = g.Shutdown(ctx); err != nil {
		log.Fatal("[ERROR] Shutdown:", err)
	} else {
		log.Println("[INFO] Exiting ..")
	}
}
