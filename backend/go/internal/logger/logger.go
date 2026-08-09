// Package logger zap 日志装配(对齐 Luna-CY 模板 internal/logger)
package logger

import "go.uber.org/zap"

// L 全局日志实例(Init 后可用);zap.L() 经 ReplaceGlobals 同步可用
var L *zap.Logger

// Init 按运行模式初始化 zap: release=JSON 生产格式,其余=彩色开发格式
func Init(mode string) error {
	var l *zap.Logger
	var err error
	if mode == "release" {
		l, err = zap.NewProduction()
	} else {
		l, err = zap.NewDevelopment()
	}
	if err != nil {
		return err
	}
	L = l
	zap.ReplaceGlobals(l)
	return nil
}

// Sync 刷新日志缓冲区,进程退出前调用
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}
