package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ruraomsk/ag-server/logger"
	"github.com/ruraomsk/camstat/dbase"
	"github.com/ruraomsk/camstat/devjson"
	"github.com/ruraomsk/camstat/devmodbus"
	"github.com/ruraomsk/camstat/send"
	"github.com/ruraomsk/camstat/setup"
	"github.com/ruraomsk/camstat/stat"
	"github.com/ruraomsk/camstat/tester"
)

var (
	//go:embed config
	config embed.FS
)
var dkset stat.DkSet
var idms chan send.MgrMessage

func init() {
	setup.Set = new(setup.Setup)
	if _, err := toml.DecodeFS(config, "config/config.toml", &setup.Set); err != nil {
		fmt.Println("Dissmis config.toml")
		os.Exit(-1)
		return
	}
	os.MkdirAll(setup.Set.LogPath, 0777)
	if err := logger.Init(setup.Set.LogPath); err != nil {
		log.Panic("Error logger system", err.Error())
		return
	}
	buffer, err := config.ReadFile("config/dktostat.json")
	if err != nil {
		logger.Error.Println(err.Error())
		fmt.Println(err.Error())
		os.Exit(-1)
		return
	}
	err = json.Unmarshal(buffer, &dkset)
	if err != nil {
		logger.Error.Println(err.Error())
		os.Exit(-1)
		return
	}

}
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Println("CamStat start")
	logger.Info.Println("CamStat start")
	chanArch, err := dbase.InitDataBase()
	if err != nil {
		logger.Error.Printf("Dbase %s", err.Error())
		return
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	idms = make(chan send.MgrMessage, 100)
	go devmodbus.Starter(&dkset, idms, chanArch)
	go devjson.Starter(&dkset, idms, chanArch)
	go send.Sender(idms)
	go tester.StartTestModbus(&dkset)
	go tester.StartTestJson(&dkset)

loop:
	for {
		<-c
		fmt.Println("Wait make abort...")
		time.Sleep(3 * time.Second)
		break loop
	}
	fmt.Println("CamStat stop")
	logger.Info.Println("CamStat stop")
}
