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

var logCfg = logconfig{Filename: os.Args[0], Level: 7, Daily: true, MaxDays: 30, Color: true}

func LogInit(dir string, filename string)  {
	os.MkdirAll(dir, 0644)

	logCfg.Filename = fmt.Sprintf("%s%c%s", dir, os.PathSeparator, filename)
	value, err := json.Marshal(&logCfg)
	if err != nil {
		panic(err.Error())
	}
	logs.Async()
	err = logs.SetLogger(logs.AdapterFile, string(value))
	if err != nil {
		panic(err.Error())
	}
}

