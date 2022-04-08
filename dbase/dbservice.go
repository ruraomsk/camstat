package dbase

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"github.com/ruraomsk/ag-server/logger"
	"github.com/ruraomsk/ag-server/pudge"
	"github.com/ruraomsk/camstat/setup"
)

var ConDB *sql.DB
var err error
var work bool
var mutex sync.Mutex
var chanArch chan pudge.ArchStat

func writerStatistics() {
	for {
		s := <-chanArch
		// fmt.Printf("write stat %v\n", s)
		WriteStat(&s)
	}
}

func InitDataBase() (chan pudge.ArchStat, error) {
	dbinfo := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		setup.Set.DataBase.Host, setup.Set.DataBase.User,
		setup.Set.DataBase.Password, setup.Set.DataBase.DBname)

	ConDB, err = sql.Open("postgres", dbinfo)
	chanArch = make(chan pudge.ArchStat, 100)
	if err != nil {
		logger.Error.Printf("Запрос на открытие %s %s", dbinfo, err.Error())
		return chanArch, err
	}
	work = true
	go writerStatistics()
	return chanArch, nil
}
func CloseDB() {
	ConDB.Close()
}
func WriteStat(rs *pudge.ArchStat) error {
	if !work {
		return fmt.Errorf("db is closed")
	}
	mutex.Lock()
	defer mutex.Unlock()
	w := fmt.Sprintf("select count(*) from public.statistics where date='%s' and region=%d and area=%d and id=%d;",
		rs.Date.Format("2006-01-02"), rs.Region, rs.Area, rs.ID)
	rows, err := ConDB.Query(w)
	if err != nil {
		return err
	}
	var count int
	for rows.Next() {
		rows.Scan(&count)
	}
	rows.Close()
	js, _ := json.Marshal(&rs)
	if count == 0 {
		w = fmt.Sprintf("INSERT INTO public.statistics(region, area, id, date, stat) VALUES (%d, %d, %d, '%s', '%s');",
			rs.Region, rs.Area, rs.ID, rs.Date.Format("2006-01-02"), string(js))
	} else {
		w = fmt.Sprintf("Update public.statistics set stat='%s' where date='%s' and region=%d and area=%d and id=%d;",
			string(js), rs.Date.Format("2006-01-02"), rs.Region, rs.Area, rs.ID)

	}
	_, err = ConDB.Exec(w)
	return err
}

func GetCross(dk pudge.Region) (pudge.Cross, error) {
	mutex.Lock()
	defer mutex.Unlock()
	w := fmt.Sprintf("select state from public.\"cross\" where region=%d and area=%d and id=%d;", dk.Region, dk.Area, dk.ID)
	rows, err := ConDB.Query(w)
	if err != nil {
		return pudge.Cross{}, err
	}
	var cross pudge.Cross
	var state []byte
	for rows.Next() {
		rows.Scan(&state)
		err := json.Unmarshal(state, &cross)
		if err != nil {
			return pudge.Cross{}, err
		}
	}
	return cross, nil
}
func GetArhs(dk pudge.Region, date time.Time) (pudge.ArchStat, error) {
	mutex.Lock()
	defer mutex.Unlock()
	w := fmt.Sprintf("select stat from public.statistics where date='%s' and region=%d and area=%d and id=%d;", date.Format("2006-01-02"), dk.Region, dk.Area, dk.ID)
	rows, err := ConDB.Query(w)
	if err != nil {
		return pudge.ArchStat{}, err
	}
	stat := pudge.ArchStat{Region: dk.Region, Area: dk.Area, ID: dk.ID, Date: date, Statistics: make([]pudge.Statistic, 0)}
	var st []byte
	for rows.Next() {
		rows.Scan(&st)
		err := json.Unmarshal(st, &stat)
		if err != nil {
			return stat, err
		}
	}
	return stat, nil
}
