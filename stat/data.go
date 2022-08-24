package stat

type DkSet struct {
	Region int      `json:"region"` //Номер региона
	Dks    []DKStat `json:"dks"`
}
type DKStat struct {
	UID  int    `json:"uid"`  //Внешной идентификатор
	Area int    `json:"area"` //Номер района
	Ndk  int    `json:"dk"`   // Номер ДК
	Type string `json:"type"`
	Port int    `json:"port"` // Номер порта где открывается Modbus
	Size int    `json:"size"` //Кол-во каналов
	Demo bool   `json:"demo"` //Истина то запускать генератор
	ID   int    `json:"-"`    // ID устройства в системе
}
