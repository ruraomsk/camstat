package stat

type DkSet struct {
	Region int      `json:"region"` //Номер региона
	Dks    []DKStat `json:"dks"`
}
type DKStat struct {
	Area int    `json:"area"` //Номер района
	Ndk  int    `json:"dk"`   // Номер ДК
	Type string `json:"type"`
	Port int    `json:"port"` // Номер порта где открывается Modbus
	ID   int    `json:"-"`    // ID устройства в системе
}
