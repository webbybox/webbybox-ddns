package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	logger "github.com/webbybox/webbybox-logger"
)

var configPath = "./ddns-config.json"

type config struct {
	SecretPath      string `json:"secretPath"`
	IntervalMinutes string `json:"intervalMinutes"`
	TargetUrl       string `json:"targetUrl"`
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

func getSecretStr(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	s := strings.TrimSpace(string(data))
	return s, nil
}

func sendReq(secretStr string, targetUrl string) {
	payload := map[string]string{
		"secret": secretStr,
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

	intervalMin, err := strconv.Atoi(cfg.IntervalMinutes)
	if err != nil || intervalMin <= 0 {
		logger.Fatalf("Invalid intervalMinutes in config: %v", cfg.IntervalMinutes)
	}

	secretStr, err := getSecretStr(cfg.SecretPath)
	if err != nil {
		logger.Fatalf("Failed to read device secret: %v", err)
	}

	// Send once on startup
	sendReq(secretStr, cfg.TargetUrl)

	ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sendReq(secretStr, cfg.TargetUrl)
	}
}
