package tester

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ruraomsk/ag-server/logger"
	"github.com/ruraomsk/camstat/modbus"
	"github.com/ruraomsk/camstat/stat"
)

type Master struct {
	connect string //Строка соединения со слайвом
	master  modbus.TCPClientHandler
	client  modbus.Client
}

func (m *Master) worker() {
	rand.Seed(time.Now().Unix())
	m.master = *modbus.NewTCPClientHandler(m.connect)
	// m.master.Logger = logger.Error
	m.master.SlaveId = 1
	m.master.Timeout = time.Second
	m.master.IdleTimeout = time.Minute
	for {
		if err := m.master.Connect(); err != nil {
			logger.Error.Printf("error modbus %s %s", m.connect, err.Error())
			time.Sleep(5 * time.Second)
			continue
		}
		m.client = modbus.NewClient(&m.master)
		oneSecond := time.NewTicker(time.Second)
		for {
			<-oneSecond.C
			bhrs := make([]byte, 0)
			t := make([]byte, 4)
			for i := 0; i < 12; i++ {
				for j := 0; j < 4; j++ {
					t[j] = byte(rand.Intn(5))
				}
				bhrs = append(bhrs, byte(t[0]<<4|t[1]))
				bhrs = append(bhrs, byte(t[2]<<4|t[3]))
			}
			_, err := m.client.WriteMultipleRegisters(0, 12, bhrs)
			if err != nil {
				logger.Error.Printf("%s %s", m.connect, err.Error())
				break
			}

		}
		m.master.Close()
		time.Sleep(5 * time.Second)
	}
}

var masters []*Master

func StartTestModbus(dkset *stat.DkSet) {
	masters = make([]*Master, 0)
	for _, v := range dkset.Dks {
		if v.Type == "modbus" {
			m := Master{connect: fmt.Sprintf("127.0.0.1:%d", v.Port)}
			go m.worker()
			masters = append(masters, &m)
		}
	}
	for {
		time.Sleep(time.Second)
	}

}
