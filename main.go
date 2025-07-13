package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	logger "github.com/webbybox/webbybox-logger"
)

var configPath = "./ddns-config.json"

type config struct {
	SecretPath      string  `json:"secretPath"`
	IntervalMinutes float32 `json:"intervalMinutes"`
	TargetUrl       string  `json:"targetUrl"`
}

func loadConfig() (*config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	cfg := &config{}
	if err := decoder.Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getDataFromSecretPath(path string) (string, error) {
	s, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(s), nil
}

func getPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(ip), nil
}

func sendReq(ip string, secret string, targetUrl string) {
	payload := map[string]string{
		"ipAddress": ip,
		"secret":    secret,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf("Failed to marshal payload: %v", err)
		return
	}

	resp, err := http.Post(targetUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		logger.Errorf("HTTP request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("Non-200 response: %s", resp.Status)
	} else {
		logger.Infof("Successfully sent DDNS update")
	}
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		logger.Fatalf("Failed to load ddns config: %v", err)
	}

	secret, err := getDataFromSecretPath(cfg.SecretPath)
	if err != nil {
		logger.Fatalf("Failed to read device secret: %v", err)
	}

	var lastIP string

	ticker := time.NewTicker(time.Duration(cfg.IntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		ip, err := getPublicIP()
		if err != nil {
			logger.Errorf("Could not retrieve public IP Address: %v", err)
			goto waitNext
		}

		if ip != lastIP {
			logger.Infof("Requesting update to Cloudflare dns record with new IP Address: %s", ip)
			sendReq(ip, secret, cfg.TargetUrl)
			lastIP = ip
		} else {
			logger.Infof("Public IP unchanged (%s), skipping update", ip)
		}

	waitNext:
		<-ticker.C
	}
}
