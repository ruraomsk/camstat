package devmodbus

import (
	"fmt"
	"time"

	"github.com/ruraomsk/ag-server/logger"
	"github.com/ruraomsk/ag-server/pudge"
	"github.com/ruraomsk/camstat/send"
	"github.com/ruraomsk/camstat/setup"
	"github.com/tbrandon/mbserver"
)

type OneSecondData struct {
	Value [16]int //кол-во ТС в зоне детектирования или скорость в зависмости от типа
	Good  bool    //Качество сигнала
}
type Device struct {
	Port       int
	ID         int   //ID устройства в системе
	Mrgs       []int //Номера каналов где следим за МГР
	Type       int   //Истина по интенсивности ложь по скорости
	Tsum       int   //Время усреднения в секундах
	Time       int   //Время последней операции в секундах от начала суток
	Intervals  map[int]OneSecondData
	server     *mbserver.Server
	stat       pudge.ArchStat
	chanArch   chan pudge.ArchStat
	isInterval bool
	chanMGR    chan send.MgrMessage
}

func (d *Device) Worker() {
	d.isInterval = false
	d.Intervals = make(map[int]OneSecondData)
	if TimeNowOfSecond()%d.Tsum == 0 {
		d.makeNewMap()
	}
	d.server = mbserver.NewServer()
	// d.server.RegisterFunctionHandler(6, writeHoldingRegister)
	// d.server.RegisterFunctionHandler(16, writeHoldingRegisters)

	con := fmt.Sprintf(":%d", d.Port)
	err := d.server.ListenTCP(con)
	if err != nil {
		logger.Error.Printf("listen %s %s", con, err.Error())
		return
	}
	// oldTime := 0
	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C
		if TimeNowOfSecond()%d.Tsum == 0 {
			// Закончился период усреднения передаем статистику
			// fmt.Printf("%6d dev %d %d make new stat\n", TimeNowOfSecond(), d.ID, d.Tsum)
			d.sendStatistics(TimeNowOfSecond())
			// Очищаем хранилище секунд
			d.makeNewMap()
			if TimeNowOfSecond() == 0 {
				//Новые сутки
				d.stat.Statistics = make([]pudge.Statistic, 0)
			}
		}
		// if oldTime != int(d.server.DiscreteInputs[0]) {
		// Есть новые данные
		d.addData(TimeNowOfSecond())
		// 	oldTime = int(d.server.DiscreteInputs[0])
		// }

		// if oldTime == TimeNowOfSecond() {
		//Готовим посылку по МГР
		d.makeMGR()
		// }
	}

}
func (d *Device) makeMGR() {
	value := d.getValue()
	mgr := send.MgrMessage{ID: d.ID, Time: time.Now(), Mgrs: make([]send.Mgr, 0)}
	for _, v := range d.Mrgs {
		if value[v-1] > 0 {
			mgr.Mgrs = append(mgr.Mgrs, send.Mgr{Chanel: v, Count: value[v-1]})
		}
	}
	if len(mgr.Mgrs) > 0 {
		d.chanMGR <- mgr
	}

}
func (d *Device) makeNewMap() {
	d.Intervals = make(map[int]OneSecondData)
	var value [16]int
	for i := 0; i < len(value); i++ {
		value[i] = 0
	}
	for i := 0; i < d.Tsum; i++ {
		d.Intervals[i] = OneSecondData{Good: false, Value: value}
	}
	d.isInterval = true
}
func (d *Device) sendStatistics(ptime int) {
	if !d.isInterval {
		return
	}
	var value [16]int
	for i := 0; i < len(value); i++ {
		value[i] = 0
	}
	g := OneSecondData{Value: value}
	countAll := 0
	countNotGood := 0

	for _, v := range d.Intervals {
		countAll++
		if v.Good {
			for i, d := range v.Value {
				g.Value[i] += d
			}
		} else {
			countNotGood++
		}
	}
	g.Good = true
	if float32(countNotGood)/float32(countAll) > setup.Set.BadProc {
		g.Good = false
	}
	s := pudge.Statistic{Period: ptime / d.Tsum, Type: d.Type, TLen: d.Tsum / 60, Hour: ptime / 3600, Min: (ptime % 3600) / 60, Datas: make([]pudge.DataStat, 0)}
	logger.Debug.Printf("id %d time %d period %d hour %d min %d ", d.ID, ptime, s.Period, s.Hour, s.Min)
	for i, v := range g.Value {
		st := 0
		if !g.Good {
			st = 1
		}
		if d.Type == 1 || d.Type == 0 {
			s.Datas = append(s.Datas, pudge.DataStat{Chanel: i + 1, Status: st, Intensiv: v})
		}
		if d.Type == 2 {
			s.Datas = append(s.Datas, pudge.DataStat{Chanel: i + 1, Status: st, Speed: v})
		}
	}
	// fmt.Printf("new add stat %v\n", s)
	d.stat.Statistics = append(d.stat.Statistics, s)
	d.stat.Date = time.Now()
	d.chanArch <- d.stat
}
func (d *Device) addData(writeTime int) {
	if !d.isInterval {
		return
	}
	t := writeTime % d.Tsum
	if _, is := d.Intervals[t]; !is {
		return
	}
	d.Intervals[t] = OneSecondData{Good: true, Value: d.getValue()}
}
func (d *Device) getValue() [16]int {
	var res [16]int
	base := 0
	if d.Type == 1 {
		for i := 0; i < 16; i++ {
			pos := int(i / 4)
			shift := (i % 4) * 4
			res[i] = int((d.server.HoldingRegisters[base+pos] >> shift) & 0xf)
		}
	}
	if d.Type == 2 {
		base = 4
		for i := 0; i < 16; i++ {
			pos := int(i / 2)
			shift := (i % 2) * 8
			res[i] = int((d.server.HoldingRegisters[base+pos] >> shift) & 0xff)
		}
	}
	return res
}
func TimeNowOfSecond() int {
	return time.Now().Hour()*3600 + time.Now().Minute()*60 + time.Now().Second()
}
