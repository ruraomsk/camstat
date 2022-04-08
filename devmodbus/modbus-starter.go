package devmodbus

import (
	"strconv"
	"strings"
	"time"

	"github.com/ruraomsk/ag-server/logger"
	"github.com/ruraomsk/ag-server/pudge"
	"github.com/ruraomsk/camstat/dbase"
	"github.com/ruraomsk/camstat/send"
	"github.com/ruraomsk/camstat/stat"
)

var err error
var devices []DeviceModbus

func Starter(dkset *stat.DkSet, idms chan send.MgrMessage, chanArch chan pudge.ArchStat) {
	devices = make([]DeviceModbus, 0)
	for _, v := range dkset.Dks {
		if v.Type == "modbus" {
			reg := pudge.Region{Region: dkset.Region, Area: v.Area, ID: v.Ndk}
			dev := DeviceModbus{size: v.Size, Port: v.Port, chanMGR: idms, chanArch: chanArch, Mrgs: make([]int, 0)}
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
			logger.Info.Printf("Создан modbus %v", reg)
			devices = append(devices, dev)
		}
	}
	for {
		time.Sleep(time.Second)
	}
}
