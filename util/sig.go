package util

import (
	"github.com/astaxie/beego/logs"
	"os"
	"os/signal"
	"syscall"
)

func WaitSignal(proc func(sig os.Signal))  {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)

	sig := <- signalChan
	logs.Warn("recv signal %s", sig.String())
	proc(sig)
	logs.Info("ready to exit")
	os.Exit(-1)
}
