package hue

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
	sse "github.com/r3labs/sse/v2"
	"github.com/spf13/viper"
)

type HueEventConsumer struct {
	Logger *log.Logger

	client       *sse.Client
	eventChannel chan *sse.Event
}

func NewHueEventConsumer(logger *log.Logger) *HueEventConsumer {
	return &HueEventConsumer{Logger: logger}
}

func (h *HueEventConsumer) Subscribe(eventChannel chan *sse.Event) {

	h.eventChannel = eventChannel
	h.client = sse.NewClient(fmt.Sprintf("https://%s/eventstream/clip/v2", viper.GetString("bridgeIp")))

	h.client.Connection.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	h.client.Headers["hue-application-key"] = viper.GetString("hueApplicationKey")

	h.client.OnConnect(func(_ *sse.Client) {
		h.Logger.Info("Connected to HUE bridge, listening for events...")
	})
	h.client.OnDisconnect(func(c *sse.Client) {
		h.Logger.Info("Disconnected from HUE bridge")
	})

	if err := h.client.SubscribeChan("", h.eventChannel); err != nil {
		h.Logger.Errorf("error subscribing to light updates: %s", err)
	}

}

func (h *HueEventConsumer) Unsubscribe() {
	h.Logger.Debug("Unsubscribe events")
	h.client.Unsubscribe(h.eventChannel)
}
