package send

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ruraomsk/ag-server/logger"
	"github.com/ruraomsk/camstat/setup"
)

type MgrMessage struct {
	ID   int       `json:"id"`
	Time time.Time `json:"time"`
	Mgrs []Mgr     `json:"mgrs"`
}
type Mgr struct {
	Chanel int `json:"ch"`
	Count  int `json:"count"`
}

var work bool

func tcpSender(ids chan MgrMessage) {
	for {
		socket, err := net.Dial("tcp", setup.Set.ConnectMgr)
		if err != nil {
			logger.Error.Printf("connect %s %s ", setup.Set.ConnectMgr, err.Error())
			time.Sleep(5 * time.Second)
			continue
		}
		writer := bufio.NewWriter(socket)
		work = true
		ticker := time.NewTimer(1 * time.Minute)
	loop:
		for {
			select {
			case s := <-ids:
				j, err := json.Marshal(s)
				if err != nil {
					logger.Error.Printf("json marshal %s", err.Error())
					continue
				}
				writer.WriteString(string(j))
				writer.WriteString("\n")
				err = writer.Flush()
				if err != nil {
					logger.Error.Printf("sender %s %s", socket.RemoteAddr().String(), err.Error())
					work = false
					break loop
				}
				ticker.Stop()
				ticker = time.NewTimer(1 * time.Minute)
			case <-ticker.C:
				writer.WriteString("0\n")
				err = writer.Flush()
				if err != nil {
					logger.Error.Printf("sender %s %s", socket.RemoteAddr().String(), err.Error())
					work = false
					break loop

				}
				ticker = time.NewTimer(1 * time.Minute)
			}
		}
		socket.Close()
	}
}
func tcpReciver() {
	ln, err := net.Listen("tcp4", fmt.Sprintf(":%d", setup.Set.PortMgr))
	if err != nil {
		logger.Error.Printf("Open MGR port %s", err.Error())
		return
	}
	for {
		socket, err := ln.Accept()
		if err != nil {
			logger.Error.Printf("Accept %s", err.Error())
			continue
		}
		logger.Info.Printf("new user MGR %s", socket.RemoteAddr().String())
		go workerMGR(socket)
	}

}
func workerMGR(socket net.Conn) {
	defer socket.Close()
	reader := bufio.NewReader(socket)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			logger.Error.Printf("Mgr user %s %s", socket.RemoteAddr().String(), err.Error())
			return
		}
		s = strings.ReplaceAll(s, "\n", "")
		// if strings.Compare(s, "0") != 0 {
		// 	logger.Debug.Printf("mgr:%s", s)
		// }
	}
}
func Sender(ids chan MgrMessage) {
	idss := make(chan MgrMessage, 100)
	go tcpReciver()
	go tcpSender(idss)
	for {
		idm := <-ids
		// logger.Debug.Printf("mgr %v", idm)
		if work {
			idss <- idm
		}
	}
}
