package hue

type HueDeviceService struct {
	RID   string `json:"rid"`
	RType string `json:"rtype"`
}

type HueDevice struct {
	Id       string `json:"id"`
	Metadata struct {
		Name string `json:"name"`
		Type string `json:"archetype"`
	} `json:"metadata"`
	Services []HueDeviceService `json:"services"`
}

type HueLight struct {
	HueDevice
	On struct {
		On bool `json:"on"`
	} `json:"on"`
}

type HueDeviceGroup struct {
	HueDevice
	Children []HueDeviceService `json:"children"`
}

type DevicesResponse struct {
	Errors []interface{} `json:"errors"`
	Data   []HueDevice   `json:"data"`
}

type LightResponse struct {
	Errors []interface{} `json:"errors"`
	Data   []HueLight    `json:"data"`
}

type GroupResponse struct {
	Errors []interface{}    `json:"errors"`
	Data   []HueDeviceGroup `json:"data"`
}
