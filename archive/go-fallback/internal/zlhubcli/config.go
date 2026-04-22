package zlhubcli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultAPIBase        = "https://api.zlhub.cn"
	defaultModel          = "doubao-seedance-2.0-fast"
	defaultImageModel     = "doubao-seedream-5.0-lite"
	defaultImageSize      = "1440x2560"
	defaultImageFormat    = "url"
	defaultRatio          = "9:16"
	defaultDuration       = 8
	defaultGenerateAudio  = true
	defaultWatermark      = false
	defaultDownloadImages = true
)

type Config struct {
	ZLHub ZLHubConfig
}

type ZLHubConfig struct {
	APIBase              string
	Model                string
	ImageModel           string
	DefaultResolution    string
	DefaultImageSize     string
	DefaultImageFormat   string
	DefaultDownloadImage bool
	CallbackURL          string
	DefaultRatio         string
	DefaultDuration      int
	DefaultGenerateAudio bool
	DefaultWatermark     bool
}

func DefaultConfig() Config {
	return Config{
		ZLHub: ZLHubConfig{
			APIBase:              defaultAPIBase,
			Model:                defaultModel,
			ImageModel:           defaultImageModel,
			DefaultImageSize:     defaultImageSize,
			DefaultImageFormat:   defaultImageFormat,
			DefaultDownloadImage: defaultDownloadImages,
			DefaultRatio:         defaultRatio,
			DefaultDuration:      defaultDuration,
			DefaultGenerateAudio: defaultGenerateAudio,
			DefaultWatermark:     defaultWatermark,
		},
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("读取配置文件失败：%w", err)
	}
	defer file.Close()

	section := ""
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripYAMLComment(scanner.Text()))
		if line == "" {
			continue
		}
		if !strings.HasPrefix(scanner.Text(), " ") && strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		if section != "zlhub" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return cfg, fmt.Errorf("配置文件第 %d 行格式错误：%s", lineNo, line)
		}
		key = strings.TrimSpace(key)
		value = normalizeYAMLValue(strings.TrimSpace(value))

		switch key {
		case "api_base":
			cfg.ZLHub.APIBase = value
		case "model":
			cfg.ZLHub.Model = value
		case "image_model":
			cfg.ZLHub.ImageModel = value
		case "default_resolution":
			cfg.ZLHub.DefaultResolution = value
		case "default_image_size":
			cfg.ZLHub.DefaultImageSize = value
		case "default_image_format":
			cfg.ZLHub.DefaultImageFormat = value
		case "default_download_image":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return cfg, fmt.Errorf("配置文件 default_download_image 必须是 true 或 false：%w", err)
			}
			cfg.ZLHub.DefaultDownloadImage = b
		case "callback_url":
			cfg.ZLHub.CallbackURL = value
		case "default_ratio":
			cfg.ZLHub.DefaultRatio = value
		case "default_duration":
			n, err := strconv.Atoi(value)
			if err != nil {
				return cfg, fmt.Errorf("配置文件 default_duration 必须是整数：%w", err)
			}
			cfg.ZLHub.DefaultDuration = n
		case "default_generate_audio":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return cfg, fmt.Errorf("配置文件 default_generate_audio 必须是 true 或 false：%w", err)
			}
			cfg.ZLHub.DefaultGenerateAudio = b
		case "default_watermark":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return cfg, fmt.Errorf("配置文件 default_watermark 必须是 true 或 false：%w", err)
			}
			cfg.ZLHub.DefaultWatermark = b
		}
	}
	if err := scanner.Err(); err != nil {
		return cfg, fmt.Errorf("读取配置文件失败：%w", err)
	}
	return cfg, nil
}

func normalizeYAMLValue(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func stripYAMLComment(line string) string {
	inSingle := false
	inDouble := false
	for i, r := range line {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return line[:i]
			}
		}
	}
	return line
}
