package devjson

import (
	"time"

	"github.com/ruraomsk/ag-server/logger"
	"github.com/ruraomsk/ag-server/pudge"
	"github.com/ruraomsk/camstat/send"
	"github.com/ruraomsk/camstat/setup"
)

type OneSecondData struct {
	Value [16]int //кол-во ТС в зоне детектирования или скорость в зависмости от типа
	Good  bool    //Качество сигнала
}
type externalData struct {
	intime int     //Число секунд от начала суток
	vTS    [16]int //кол-во ТС в зоне детектирования
	sTS    [16]int //скорость ТС в зоне детектирования
}
type DeviceJson struct {
	Port       int
	ID         int   //ID устройства в системе
	Mrgs       []int //Номера каналов где следим за МГР
	Type       int   //Истина по интенсивности ложь по скорости
	Tsum       int   //Время усреднения в секундах
	Intervals  map[int]OneSecondData
	chanData   chan externalData
	stat       pudge.ArchStat
	chanArch   chan pudge.ArchStat
	isInterval bool
	chanMGR    chan send.MgrMessage
	size       int //Количество каналов в статистике
}

func (d *DeviceJson) Worker() {
	d.isInterval = false
	d.Intervals = make(map[int]OneSecondData)
	if TimeNowOfSecond()%d.Tsum == 0 {
		d.makeNewMap()
	}
	// d.chanData = make(chan externalData, 100)
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			if TimeNowOfSecond()%d.Tsum == 0 {
				// Закончился период усреднения передаем статистику
				// fmt.Printf("%6d dev %6d %6d make new stat\n", TimeNowOfSecond(), d.ID, d.Tsum)
				d.sendStatistics(TimeNowOfSecond())
				// Очищаем хранилище секунд
				d.makeNewMap()
				if TimeNowOfSecond() == 0 {
					//Новые сутки
					d.stat.Statistics = make([]pudge.Statistic, 0)
				}
			}
		// }
		case message := <-d.chanData:
			// Есть новые данные
			d.addData(message)
			//Готовим посылку по МГР
			d.makeMGR(message)
			// logger.Debug.Print(d.getValue(message))

		}
	}

}
func (d *DeviceJson) makeMGR(message externalData) {
	value := d.getValue(message)
	mgr := send.MgrMessage{ID: d.ID, Time: time.Now(), Mgrs: make([]send.Mgr, 0)}
	if len(d.Mrgs) == 0 {
		return
	}
	for _, v := range d.Mrgs {
		if value[v-1] > 0 {
			mgr.Mgrs = append(mgr.Mgrs, send.Mgr{Chanel: v, Count: value[v-1]})
		}
	}
	if len(mgr.Mgrs) > 0 {
		d.chanMGR <- mgr
	}

}
func (d *DeviceJson) makeNewMap() {
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
func (d *DeviceJson) sendStatistics(ptime int) {
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
	s := pudge.Statistic{Period: ptime / d.Tsum, Type: d.Type, TLen: d.Tsum / 60, Hour: ptime / 3600, Min: (ptime % 3600) / 60,
		Datas: make([]pudge.DataStat, 0)}
	// logger.Debug.Printf("id %d time %d period %d hour %d min %d ", d.ID, ptime, s.Period, s.Hour, s.Min)
	for i, v := range g.Value {
		if i >= d.size {
			continue
		}
		st := 0
		if !g.Good {
			st = 1
		}
		if d.Type == 2 {
			s.Datas = append(s.Datas, pudge.DataStat{Chanel: i + 1, Status: st, Speed: v})
		} else {
			s.Datas = append(s.Datas, pudge.DataStat{Chanel: i + 1, Status: st, Intensiv: v})
		}
	}
	// logger.Debug.Printf("new add stat %v\n", s)
	d.stat.Statistics = append(d.stat.Statistics, s)
	d.stat.Date = time.Now()
	d.chanArch <- d.stat
}
func (d *DeviceJson) addData(message externalData) {
	if !d.isInterval {
		return
	}
	t := message.intime % d.Tsum
	if _, is := d.Intervals[t]; !is {
		logger.Error.Printf("not %d", message.intime)
		return
	}
	d.Intervals[t] = OneSecondData{Good: true, Value: d.getValue(message)}
	// logger.Debug.Print(d.Intervals[t])
}
func (d *DeviceJson) getValue(message externalData) [16]int {
	if d.Type == 1 {
		return message.vTS
	}
	if d.Type == 2 {
		return message.sTS
	}
	return message.vTS
}
func TimeNowOfSecond() int {
	return time.Now().Hour()*3600 + time.Now().Minute()*60 + time.Now().Second()
}
func TimeToSeconds(p time.Time) int {
	return p.Hour()*3600 + p.Minute()*60 + p.Second()
}
