package setup

var Set *Setup

type Setup struct {
	LogPath    string   `toml:"log"`
	DataBase   DataBase `toml:"dataBase"`
	ConnectMgr string   `toml:"connectmgr"`
	PortMgr    int      `toml:"portmgr"`
	BadProc    float32  `toml:"badproc"` //Процент пропущенных записй после которого весть интервал считаетсая плохим
}

//DataBase настройки базы данных postresql
type DataBase struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	DBname   string `toml:"dbname"`
}
