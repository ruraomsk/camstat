package devjson

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ruraomsk/ag-server/logger"
	"github.com/ruraomsk/ag-server/pudge"
	"github.com/ruraomsk/camstat/dbase"
	"github.com/ruraomsk/camstat/send"
	"github.com/ruraomsk/camstat/setup"
	"github.com/ruraomsk/camstat/stat"
)

var err error
var devices map[pudge.Region]DeviceJson
var uids map[int]pudge.Region

func Starter(dkset *stat.DkSet, idms chan send.MgrMessage, chanArch chan pudge.ArchStat) {

	devices = make(map[pudge.Region]DeviceJson)
	uids = make(map[int]pudge.Region)

	for _, v := range dkset.Dks {
		if v.Type == "json" {
			reg := pudge.Region{Region: dkset.Region, Area: v.Area, ID: v.Ndk}
			dev := DeviceJson{size: v.Size, Port: v.Port,
				chanMGR: idms, chanArch: chanArch, Mrgs: make([]int, 0), chanData: make(chan externalData)}
			dev.stat, err = dbase.GetArhs(reg, time.Now())
			if err != nil {
				logger.Error.Printf("чтение статистики %v %s", reg, err.Error())
				return
			}
			cr, err := dbase.GetCross(reg)
			if err != nil {
				logger.Error.Printf("чтение cross %v %s", reg, err.Error())
				return
			}
			dev.ID = cr.IDevice
			for _, v := range cr.Arrays.SetTimeUse.Uses {
				if v.Tvps == 3 {
					if strings.Contains(v.Name, "вх") {
						v.Name = strings.ReplaceAll(v.Name, "вх", "")
						i, _ := strconv.Atoi(v.Name)
						if i > 0 && i <= 16 {
							dev.Mrgs = append(dev.Mrgs, i)
						}
					}
				}
			}
			//For Debug
			for i := 1; i < 3; i++ {
				dev.Mrgs = append(dev.Mrgs, i)
			}
			if len(cr.Arrays.StatDefine.Levels) != 0 {
				dev.Type = cr.Arrays.StatDefine.Levels[0].TypeSt
				dev.Tsum = cr.Arrays.StatDefine.Levels[0].Period * 60
			} else {
				dev.Type = 1
				dev.Tsum = 300
			}
			if dev.Type != 1 && dev.Type != 2 {
				dev.Type = 1
			}
			go dev.Worker()
			logger.Info.Printf("Создан devicejson %v", reg)
			devices[reg] = dev
			uids[v.UID] = reg
		}
	}
	ln, err := net.Listen("tcp4", fmt.Sprintf(":%d", setup.Set.JsonStat.PortJson))
	if err != nil {
		logger.Error.Printf("Open JSON port %s", err.Error())
		return
	}
	for {
		socket, err := ln.Accept()
		if err != nil {
			logger.Error.Printf("Accept %s", err.Error())
			continue
		}
		logger.Info.Printf("new user JSON %s", socket.RemoteAddr().String())
		go workerStat(socket)
	}

}

type EntryMessage struct {
	Uid     int   `json:"uid"` //Внешний идентификатор
	CountTC []int `json:"cts"` //начилие по каждой зоне детектирования число ТС
	SpeedTC []int `json:"stc"` //начилие по каждой зоне детектирования скорость ТС
	//Зоны заполняются от первой до последней максимум 16
}

func workerStat(socket net.Conn) {
	defer socket.Close()
	reader := bufio.NewReader(socket)
	var ext EntryMessage
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			logger.Error.Printf("Statistic user %s %s", socket.RemoteAddr().String(), err.Error())
			return
		}
		s = strings.ReplaceAll(s, "\n", "")
		// logger.Debug.Printf("%s==%s", socket.RemoteAddr().String(), s)

		err = json.Unmarshal([]byte(s), &ext)
		if err != nil {
			logger.Error.Printf("Statistic user %s ison unmarhal  %s", socket.RemoteAddr().String(), err.Error())
			return
		}
		// logger.Debug.Printf("%s->%v", socket.RemoteAddr().String(), ext)
		nm := externalData{intime: TimeNowOfSecond()}
		for i, v := range ext.CountTC {
			nm.vTS[i] = v
		}
		for i, v := range ext.SpeedTC {
			nm.sTS[i] = v
		}
		reg, is := uids[ext.Uid]
		if !is {
			logger.Error.Printf("нет внешнего %d", ext.Uid)
			continue
		}
		// logger.Debug.Print(nm)
		devices[reg].chanData <- nm

	}
}
