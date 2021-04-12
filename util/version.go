package util

import "github.com/astaxie/beego/logs"

func VersionGet() string {
	return "v0.1.2"
}

func init()  {
	logs.Info("version:", VersionGet())
}
