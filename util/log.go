package util


import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"os"
)

type logconfig struct {
	Filename string  `json:"filename"`
	Level    int     `json:"level"`
	MaxLines int     `json:"maxlines"`
	MaxSize  int     `json:"maxsize"`
	Daily    bool    `json:"daily"`
	MaxDays  int     `json:"maxdays"`
	Color    bool    `json:"color"`
}

var logCfg = logconfig{
	Filename: os.Args[0],
	Level: logs.LevelInformational,
	Daily: true,
	MaxSize: 10*1024*1024,
	MaxLines: 100*1024,
	MaxDays: 7,
	Color: false,
}

func LogInit(dir string, debug bool, filename string)  {
	os.MkdirAll(dir, 0644)

	logCfg.Filename = fmt.Sprintf("%s%c%s", dir, os.PathSeparator, filename)
	value, err := json.Marshal(&logCfg)
	if err != nil {
		panic(err.Error())
	}
	if debug {
		err = logs.SetLogger(logs.AdapterConsole)
	} else {
		err = logs.SetLogger(logs.AdapterFile, string(value))
	}
	if err != nil {
		panic(err.Error())
	}
	logs.Async(100)
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
}

