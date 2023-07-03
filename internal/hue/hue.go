package hue

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/wheelibin/hugh/internal/config"
)

type HueAPIService struct {
	cfg    config.Config
	logger *log.Logger
}

func NewHueAPIService(cfg config.Config, logger *log.Logger) *HueAPIService {
	return &HueAPIService{cfg, logger}
}

func (h HueAPIService) GET(url string) ([]byte, error) {
	return h.makeRequest("GET", url, nil)
}

func (h HueAPIService) PUT(url string, body []byte) ([]byte, error) {
	return h.makeRequest("PUT", url, body)
}

func (h HueAPIService) makeRequest(verb string, url string, body []byte) ([]byte, error) {

	bodyReader := bytes.NewReader(body)
	req, err := http.NewRequest(verb, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// set headers
	req.Header.Set("hue-application-key", h.cfg.HueAppKey)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// make the request
	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error(err)
		return nil, err
	}

	switch resp.StatusCode {
	case 200:
		// all good
		responseBody, _ := ioutil.ReadAll(resp.Body)
		return responseBody, nil
	case 207:
		return nil, errors.New("unreachable")
	default:
		h.logger.Error("Error making Hue API call", "url", url, "status", resp.Status)
		return nil, err
	}

}
