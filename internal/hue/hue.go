package hue

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

type HueAPIService struct {
	logger *log.Logger
}

func NewHueAPIService(logger *log.Logger) *HueAPIService {
	return &HueAPIService{logger}
}

func (h HueAPIService) GET(url string) ([]byte, error) {
	return h.makeRequest("GET", url, nil)
}

func (h HueAPIService) PUT(url string, body []byte) ([]byte, error) {
	return h.makeRequest("PUT", url, body)
}

func (h HueAPIService) makeRequest(verb string, url string, body []byte) ([]byte, error) {

	bodyReader := bytes.NewReader(body)
	req, err := http.NewRequest(verb, fmt.Sprintf("https://%s%s", viper.GetString("bridgeIp"), url), bodyReader)
	if err != nil {
		return nil, err
	}

	// set headers
	req.Header.Set("hue-application-key", viper.GetString("hueApplicationKey"))
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
