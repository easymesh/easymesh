package util

import "github.com/astaxie/beego/logs"

func VersionGet() string {
	return "v0.1.1"
}

func init()  {
	logs.Info("version:", VersionGet())
}
