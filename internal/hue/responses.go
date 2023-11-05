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
	Owner struct {
		DeviceID string `json:"rid"`
	} `json:"owner"`
	ColorTemperature struct {
		MirekBounds struct {
			Min int `json:"mirek_minimum"`
			Max int `json:"mirek_maximum"`
		} `json:"mirek_schema"`
	} `json:"color_temperature"`
}

type HueDeviceGroup struct {
	HueDevice
	Children []HueDeviceService `json:"children"`
}

type HueSceneAction struct {
	Target HueDeviceService `json:"target"`
	Action struct {
		On *struct {
			On bool `json:"on"`
		} `json:"on"`
		Dimming *struct {
			Brightness float64 `json:"brightness"`
		} `json:"dimming"`
		ColorTemperature *struct {
			Mirek int `json:"mirek"`
		} `json:"color_temperature"`
	} `json:"action"`
}

type HueScene struct {
	HueDevice
	Actions []HueSceneAction `json:"actions"`
}

type DevicesResponse struct {
	Errors []any       `json:"errors"`
	Data   []HueDevice `json:"data"`
}

type LightResponse struct {
	Errors []any      `json:"errors"`
	Data   []HueLight `json:"data"`
}

type GroupResponse struct {
	Errors []any            `json:"errors"`
	Data   []HueDeviceGroup `json:"data"`
}

type SceneResponse struct {
	Errors []any      `json:"errors"`
	Data   []HueScene `json:"data"`
}
