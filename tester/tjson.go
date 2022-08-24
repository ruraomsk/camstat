package tester

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/ruraomsk/ag-server/logger"
	"github.com/ruraomsk/camstat/devjson"
	"github.com/ruraomsk/camstat/setup"
	"github.com/ruraomsk/camstat/stat"
)

type MasterJson struct {
	connect string //Строка соединения со слайвом
	size    int
	uid     int
}

func (m *MasterJson) worker() {
	rand.Seed(time.Now().Unix())
	for {
		socket, err := net.Dial("tcp", m.connect)
		if err != nil {
			logger.Error.Printf("error json %s %s", m.connect, err.Error())
			time.Sleep(5 * time.Second)
			continue
		}
		writer := bufio.NewWriter(socket)
		oneSecond := time.NewTicker(time.Second)
		message := devjson.EntryMessage{Uid: m.uid}
		for {
			<-oneSecond.C
			message.CountTC = make([]int, 0)
			message.SpeedTC = make([]int, 0)
			for i := 0; i < m.size; i++ {
				message.CountTC = append(message.CountTC, rand.Intn(5))
				message.SpeedTC = append(message.SpeedTC, rand.Intn(30))
			}
			buffer, err := json.Marshal(&message)
			if err != nil {
				logger.Error.Print(err.Error())
				break
			}
			writer.WriteString(string(buffer))
			writer.WriteString("\n")
			err = writer.Flush()
			if err != nil {
				logger.Error.Print(err.Error())
				break
			}
		}
		socket.Close()
		time.Sleep(5 * time.Second)
	}
}

var mjsons []*MasterJson

func StartTestJson(dkset *stat.DkSet) {
	mjsons = make([]*MasterJson, 0)
	for _, v := range dkset.Dks {
		if v.Type == "json" && v.Demo {
			m := MasterJson{connect: fmt.Sprintf("127.0.0.1:%d", setup.Set.JsonStat.PortJson), size: v.Size, uid: v.UID}
			go m.worker()
			mjsons = append(mjsons, &m)
		}
	}
	for {
		time.Sleep(time.Second)
	}

}
