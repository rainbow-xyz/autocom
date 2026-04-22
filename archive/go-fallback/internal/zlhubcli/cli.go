package zlhubcli

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type App struct {
	Out        io.Writer
	Err        io.Writer
	Getenv     func(string) string
	HTTPClient *http.Client
	Now        func() time.Time
}

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type createOptions struct {
	configPath    string
	apiBase       string
	traceID       string
	file          string
	prompt        string
	images        stringList
	videos        stringList
	audios        stringList
	callbackURL   string
	model         string
	resolution    string
	ratio         string
	duration      int
	generateAudio bool
	watermark     bool
	outDir        string
}

type getOptions struct {
	configPath string
	apiBase    string
	traceID    string
	id         string
	outDir     string
}

type imageOptions struct {
	configPath     string
	apiBase        string
	traceID        string
	file           string
	prompt         string
	model          string
	size           string
	responseFormat string
	watermark      bool
	download       bool
	outDir         string
}

type taskSummary struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	VideoURL  string `json:"video_url"`
	Error     any    `json:"error"`
	UpdatedAt string `json:"updated_at"`
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		a.printRootUsage()
		return fmt.Errorf("缺少命令")
	}
	if args[0] != "zlhub" {
		a.printRootUsage()
		return fmt.Errorf("未知命令：%s", args[0])
	}
	if len(args) < 2 {
		a.printZLHubUsage()
		return fmt.Errorf("缺少 zlhub 子命令")
	}

	switch args[1] {
	case "create":
		return a.runCreate(args[2:])
	case "get":
		return a.runGet(args[2:])
	case "image":
		return a.runImage(args[2:])
	default:
		a.printZLHubUsage()
		return fmt.Errorf("未知 zlhub 子命令：%s", args[1])
	}
}

func (a *App) runCreate(args []string) error {
	var opts createOptions
	opts.configPath = "autocom.yaml"
	opts.duration = -1

	fs := flag.NewFlagSet("autocom zlhub create", flag.ContinueOnError)
	fs.SetOutput(a.Err)
	fs.StringVar(&opts.configPath, "config", opts.configPath, "项目配置文件路径")
	fs.StringVar(&opts.apiBase, "api-base", "", "ZLHub API Base")
	fs.StringVar(&opts.traceID, "trace-id", "", "请求追踪 ID")
	fs.StringVar(&opts.file, "file", "", "完整请求体 JSON 文件")
	fs.StringVar(&opts.prompt, "prompt", "", "视频提示词")
	fs.Var(&opts.images, "image", "图片素材，格式 role=url，可重复")
	fs.Var(&opts.videos, "video", "视频素材，格式 role=url，可重复")
	fs.Var(&opts.audios, "audio", "音频素材，格式 role=url，可重复")
	fs.StringVar(&opts.callbackURL, "callback-url", "", "任务状态回调地址")
	fs.StringVar(&opts.model, "model", "", "视频生成模型")
	fs.StringVar(&opts.resolution, "resolution", "", "视频分辨率，例如 480p 或 720p")
	fs.StringVar(&opts.ratio, "ratio", "", "视频比例，例如 9:16 或 16:9")
	fs.IntVar(&opts.duration, "duration", opts.duration, "视频时长，单位秒")
	fs.BoolVar(&opts.generateAudio, "generate-audio", false, "是否生成音频")
	fs.BoolVar(&opts.watermark, "watermark", false, "是否添加水印")
	fs.StringVar(&opts.outDir, "out-dir", "", "任务输出目录")
	if err := fs.Parse(args); err != nil {
		return err
	}
	visited := visitedFlags(fs)

	cfg, err := LoadConfig(opts.configPath)
	if err != nil {
		return err
	}
	apiKey := strings.TrimSpace(a.getenv("ZLHUB_API_KEY"))
	if apiKey == "" {
		return fmt.Errorf("缺少环境变量 ZLHUB_API_KEY，请先在宿主机环境变量中配置 API Key")
	}
	if strings.TrimSpace(opts.outDir) == "" {
		return fmt.Errorf("缺少 --out-dir，请指定任务输出目录")
	}

	payload, err := buildCreatePayload(opts, cfg, visited)
	if err != nil {
		return err
	}

	zlhubDir := filepath.Join(opts.outDir, "zlhub")
	if err := os.MkdirAll(zlhubDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败：%w", err)
	}
	if err := writeJSONFile(filepath.Join(zlhubDir, "request.json"), payload); err != nil {
		return err
	}

	traceID := opts.traceID
	if traceID == "" {
		traceID = a.makeTraceID()
	}
	apiBase := chooseString(opts.apiBase, cfg.ZLHub.APIBase, defaultAPIBase)
	body, statusCode, err := a.postJSON(context.Background(), joinAPI(apiBase, "/v1/task/create"), apiKey, traceID, payload)
	if writeErr := os.WriteFile(filepath.Join(zlhubDir, "create_response.json"), body, 0o644); writeErr != nil {
		return fmt.Errorf("写入创建响应失败：%w", writeErr)
	}
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("创建视频任务失败，HTTP 状态码：%d", statusCode)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("创建响应不是合法 JSON：%w", err)
	}
	result := responsePayload(response)
	taskID, _ := result["id"].(string)
	if taskID == "" {
		return fmt.Errorf("创建响应缺少任务 ID 字段 id")
	}

	summary := taskSummary{
		TaskID:    taskID,
		Status:    chooseResponseStatus(result, "submitted"),
		VideoURL:  extractVideoURL(result),
		Error:     chooseResponseError(response, result),
		UpdatedAt: a.now().Format(time.RFC3339),
	}
	if err := writeJSONFile(filepath.Join(zlhubDir, "task.json"), summary); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "创建成功，任务 ID：%s\n", taskID)
	return nil
}

func (a *App) runGet(args []string) error {
	var opts getOptions
	opts.configPath = "autocom.yaml"

	fs := flag.NewFlagSet("autocom zlhub get", flag.ContinueOnError)
	fs.SetOutput(a.Err)
	fs.StringVar(&opts.configPath, "config", opts.configPath, "项目配置文件路径")
	fs.StringVar(&opts.apiBase, "api-base", "", "ZLHub API Base")
	fs.StringVar(&opts.traceID, "trace-id", "", "请求追踪 ID")
	fs.StringVar(&opts.id, "id", "", "ZLHub 任务 ID")
	fs.StringVar(&opts.outDir, "out-dir", "", "任务输出目录")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := LoadConfig(opts.configPath)
	if err != nil {
		return err
	}
	apiKey := strings.TrimSpace(a.getenv("ZLHUB_API_KEY"))
	if apiKey == "" {
		return fmt.Errorf("缺少环境变量 ZLHUB_API_KEY，请先在宿主机环境变量中配置 API Key")
	}
	if strings.TrimSpace(opts.id) == "" {
		return fmt.Errorf("缺少 --id，请传入 ZLHub 任务 ID")
	}
	if strings.TrimSpace(opts.outDir) == "" {
		return fmt.Errorf("缺少 --out-dir，请指定任务输出目录")
	}

	zlhubDir := filepath.Join(opts.outDir, "zlhub")
	if err := os.MkdirAll(zlhubDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败：%w", err)
	}

	traceID := opts.traceID
	if traceID == "" {
		traceID = a.makeTraceID()
	}
	apiBase := chooseString(opts.apiBase, cfg.ZLHub.APIBase, defaultAPIBase)
	endpoint := joinAPI(apiBase, "/v1/task/get/"+url.PathEscape(opts.id))
	body, statusCode, err := a.getJSON(context.Background(), endpoint, apiKey, traceID)
	if writeErr := os.WriteFile(filepath.Join(zlhubDir, "query_response.json"), body, 0o644); writeErr != nil {
		return fmt.Errorf("写入查询响应失败：%w", writeErr)
	}
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("查询视频任务失败，HTTP 状态码：%d", statusCode)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("查询响应不是合法 JSON：%w", err)
	}
	result := responsePayload(response)
	taskID, _ := result["id"].(string)
	if taskID == "" {
		taskID = opts.id
	}
	status := chooseResponseStatus(result, "unknown")
	summary := taskSummary{
		TaskID:    taskID,
		Status:    status,
		VideoURL:  extractVideoURL(result),
		Error:     chooseResponseError(response, result),
		UpdatedAt: a.now().Format(time.RFC3339),
	}
	if err := writeJSONFile(filepath.Join(zlhubDir, "task.json"), summary); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "查询成功，任务 ID：%s，状态：%s\n", taskID, status)
	return nil
}

func (a *App) runImage(args []string) error {
	var opts imageOptions
	opts.configPath = "autocom.yaml"

	fs := flag.NewFlagSet("autocom zlhub image", flag.ContinueOnError)
	fs.SetOutput(a.Err)
	fs.StringVar(&opts.configPath, "config", opts.configPath, "项目配置文件路径")
	fs.StringVar(&opts.apiBase, "api-base", "", "ZLHub API Base")
	fs.StringVar(&opts.traceID, "trace-id", "", "请求追踪 ID")
	fs.StringVar(&opts.file, "file", "", "完整图片请求体 JSON 文件")
	fs.StringVar(&opts.prompt, "prompt", "", "图片提示词")
	fs.StringVar(&opts.model, "model", "", "图片生成模型")
	fs.StringVar(&opts.size, "size", "", "图片尺寸，例如 1440x2560")
	fs.StringVar(&opts.responseFormat, "response-format", "", "响应格式，默认 url")
	fs.BoolVar(&opts.watermark, "watermark", false, "是否添加水印")
	fs.BoolVar(&opts.download, "download", true, "是否自动下载返回的图片 URL")
	fs.StringVar(&opts.outDir, "out-dir", "", "任务输出目录")
	if err := fs.Parse(args); err != nil {
		return err
	}
	visited := visitedFlags(fs)

	cfg, err := LoadConfig(opts.configPath)
	if err != nil {
		return err
	}
	apiKey := strings.TrimSpace(a.getenv("ZLHUB_API_KEY"))
	if apiKey == "" {
		return fmt.Errorf("缺少环境变量 ZLHUB_API_KEY，请先在宿主机环境变量中配置 API Key")
	}
	if strings.TrimSpace(opts.outDir) == "" {
		return fmt.Errorf("缺少 --out-dir，请指定任务输出目录")
	}

	payload, err := buildImagePayload(opts, cfg, visited)
	if err != nil {
		return err
	}

	imageDir := filepath.Join(opts.outDir, "zlhub", "image")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败：%w", err)
	}
	if err := writeJSONFile(filepath.Join(imageDir, "request.json"), payload); err != nil {
		return err
	}

	traceID := opts.traceID
	if traceID == "" {
		traceID = a.makeTraceID()
	}
	apiBase := chooseString(opts.apiBase, cfg.ZLHub.APIBase, defaultAPIBase)
	body, statusCode, err := a.postJSON(context.Background(), joinAPI(apiBase, "/v1/images/generations"), apiKey, traceID, payload)
	if writeErr := os.WriteFile(filepath.Join(imageDir, "response.json"), body, 0o644); writeErr != nil {
		return fmt.Errorf("写入图片响应失败：%w", writeErr)
	}
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("生成图片失败，HTTP 状态码：%d", statusCode)
	}

	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("图片响应不是合法 JSON：%w", err)
	}
	summary := buildImageSummary(response, a.now())
	if err := writeJSONFile(filepath.Join(imageDir, "summary.json"), summary); err != nil {
		return err
	}

	download := opts.download
	if !visited["download"] {
		download = cfg.ZLHub.DefaultDownloadImage
	}
	if download {
		if err := a.downloadImageURLs(context.Background(), summary.Images, imageDir); err != nil {
			return err
		}
	}
	fmt.Fprintf(a.Out, "图片生成成功，共 %d 张\n", len(summary.Images))
	return nil
}

func buildCreatePayload(opts createOptions, cfg Config, visited map[string]bool) (map[string]any, error) {
	var payload map[string]any
	if opts.file != "" {
		raw, err := os.ReadFile(opts.file)
		if err != nil {
			return nil, fmt.Errorf("读取请求 JSON 文件失败：%w", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("请求 JSON 文件格式错误：%w", err)
		}
	} else {
		if strings.TrimSpace(opts.prompt) == "" {
			return nil, fmt.Errorf("缺少 --prompt；如果要使用完整请求体，请传 --file request.json")
		}
		content := []any{
			map[string]any{
				"type": "text",
				"text": opts.prompt,
			},
		}
		items, err := buildAssetItems("image_url", "image_url", opts.images)
		if err != nil {
			return nil, err
		}
		content = append(content, items...)
		items, err = buildAssetItems("video_url", "video_url", opts.videos)
		if err != nil {
			return nil, err
		}
		content = append(content, items...)
		items, err = buildAssetItems("audio_url", "audio_url", opts.audios)
		if err != nil {
			return nil, err
		}
		content = append(content, items...)

		payload = map[string]any{
			"content": content,
		}
	}

	applyStringField(payload, "model", opts.model, cfg.ZLHub.Model, defaultModel, visited["model"])
	applyOptionalStringField(payload, "resolution", opts.resolution, cfg.ZLHub.DefaultResolution, visited["resolution"])
	applyStringField(payload, "ratio", opts.ratio, cfg.ZLHub.DefaultRatio, defaultRatio, visited["ratio"])
	applyIntField(payload, "duration", opts.duration, cfg.ZLHub.DefaultDuration, defaultDuration, visited["duration"])
	applyBoolField(payload, "generate_audio", opts.generateAudio, cfg.ZLHub.DefaultGenerateAudio, defaultGenerateAudio, visited["generate-audio"])
	applyBoolField(payload, "watermark", opts.watermark, cfg.ZLHub.DefaultWatermark, defaultWatermark, visited["watermark"])

	if visited["callback-url"] {
		if strings.TrimSpace(opts.callbackURL) != "" {
			payload["callback_url"] = opts.callbackURL
		} else {
			delete(payload, "callback_url")
		}
	} else if _, ok := payload["callback_url"]; !ok && strings.TrimSpace(cfg.ZLHub.CallbackURL) != "" {
		payload["callback_url"] = cfg.ZLHub.CallbackURL
	}

	return payload, nil
}

func buildImagePayload(opts imageOptions, cfg Config, visited map[string]bool) (map[string]any, error) {
	var payload map[string]any
	if opts.file != "" {
		raw, err := os.ReadFile(opts.file)
		if err != nil {
			return nil, fmt.Errorf("读取图片请求 JSON 文件失败：%w", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("图片请求 JSON 文件格式错误：%w", err)
		}
	} else {
		if strings.TrimSpace(opts.prompt) == "" {
			return nil, fmt.Errorf("缺少 --prompt；如果要使用完整图片请求体，请传 --file request.json")
		}
		payload = map[string]any{
			"prompt": opts.prompt,
		}
	}

	applyStringField(payload, "model", opts.model, cfg.ZLHub.ImageModel, defaultImageModel, visited["model"])
	applyStringField(payload, "size", opts.size, cfg.ZLHub.DefaultImageSize, defaultImageSize, visited["size"])
	applyStringField(payload, "response_format", opts.responseFormat, cfg.ZLHub.DefaultImageFormat, defaultImageFormat, visited["response-format"])
	applyBoolField(payload, "watermark", opts.watermark, cfg.ZLHub.DefaultWatermark, defaultWatermark, visited["watermark"])
	return payload, nil
}

func buildAssetItems(itemType, field string, values []string) ([]any, error) {
	items := make([]any, 0, len(values))
	for _, value := range values {
		role, assetURL, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(role) == "" || strings.TrimSpace(assetURL) == "" {
			return nil, fmt.Errorf("素材参数格式错误：%s，正确格式为 role=url", value)
		}
		items = append(items, map[string]any{
			"type": itemType,
			field: map[string]any{
				"url": strings.TrimSpace(assetURL),
			},
			"role": strings.TrimSpace(role),
		})
	}
	return items, nil
}

func applyStringField(payload map[string]any, key, cliValue, cfgValue, defaultValue string, cliSet bool) {
	if cliSet {
		if strings.TrimSpace(cliValue) != "" {
			payload[key] = cliValue
		}
		return
	}
	if _, ok := payload[key]; ok {
		return
	}
	payload[key] = chooseString("", cfgValue, defaultValue)
}

func applyOptionalStringField(payload map[string]any, key, cliValue, cfgValue string, cliSet bool) {
	if cliSet {
		if strings.TrimSpace(cliValue) != "" {
			payload[key] = cliValue
		} else {
			delete(payload, key)
		}
		return
	}
	if _, ok := payload[key]; ok {
		return
	}
	if strings.TrimSpace(cfgValue) != "" {
		payload[key] = cfgValue
	}
}

func applyIntField(payload map[string]any, key string, cliValue, cfgValue, defaultValue int, cliSet bool) {
	if cliSet {
		payload[key] = cliValue
		return
	}
	if _, ok := payload[key]; ok {
		return
	}
	if cfgValue != 0 {
		payload[key] = cfgValue
		return
	}
	payload[key] = defaultValue
}

func applyBoolField(payload map[string]any, key string, cliValue, cfgValue, defaultValue bool, cliSet bool) {
	if cliSet {
		payload[key] = cliValue
		return
	}
	if _, ok := payload[key]; ok {
		return
	}
	if cfgValue != defaultValue {
		payload[key] = cfgValue
		return
	}
	payload[key] = defaultValue
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func chooseString(cliValue, cfgValue, defaultValue string) string {
	if strings.TrimSpace(cliValue) != "" {
		return cliValue
	}
	if strings.TrimSpace(cfgValue) != "" {
		return cfgValue
	}
	return defaultValue
}

func joinAPI(apiBase, path string) string {
	return strings.TrimRight(apiBase, "/") + path
}

func (a *App) postJSON(ctx context.Context, endpoint, apiKey, traceID string, payload map[string]any) ([]byte, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("序列化请求体失败：%w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("创建 HTTP 请求失败：%w", err)
	}
	return a.doJSON(req, apiKey, traceID)
}

func (a *App) getJSON(ctx context.Context, endpoint, apiKey, traceID string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("创建 HTTP 请求失败：%w", err)
	}
	return a.doJSON(req, apiKey, traceID)
}

func (a *App) downloadImageURLs(ctx context.Context, images []imageResult, outDir string) error {
	for i, image := range images {
		if strings.TrimSpace(image.URL) == "" {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, image.URL, nil)
		if err != nil {
			return fmt.Errorf("创建图片下载请求失败：%w", err)
		}
		client := a.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("下载图片失败：%w", err)
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				err = fmt.Errorf("下载图片失败，HTTP 状态码：%d", resp.StatusCode)
				return
			}
			ext := imageFileExt(image.URL, resp.Header.Get("Content-Type"))
			path := filepath.Join(outDir, fmt.Sprintf("image_%d%s", i+1, ext))
			var file *os.File
			file, err = os.Create(path)
			if err != nil {
				err = fmt.Errorf("创建图片文件失败：%w", err)
				return
			}
			defer file.Close()
			if _, err = io.Copy(file, resp.Body); err != nil {
				err = fmt.Errorf("写入图片文件失败：%w", err)
				return
			}
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) doJSON(req *http.Request, apiKey, traceID string) ([]byte, int, error) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	if traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}
	client := a.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求 ZLHub 失败：%w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("读取 ZLHub 响应失败：%w", err)
	}
	return body, resp.StatusCode, nil
}

func chooseResponseStatus(response map[string]any, fallback string) string {
	if status, ok := response["status"].(string); ok && status != "" {
		return status
	}
	return fallback
}

func extractVideoURL(response map[string]any) string {
	for _, path := range [][]string{
		{"video_url"},
		{"content", "video_url"},
		{"output", "video_url"},
		{"result", "video_url"},
	} {
		if value := extractString(response, path...); value != "" {
			return value
		}
	}
	return ""
}

type imageSummary struct {
	Model           string        `json:"model"`
	GeneratedImages int           `json:"generated_images"`
	TotalTokens     int           `json:"total_tokens"`
	Images          []imageResult `json:"images"`
	UpdatedAt       string        `json:"updated_at"`
}

type imageResult struct {
	URL  string `json:"url,omitempty"`
	Size string `json:"size,omitempty"`
}

func buildImageSummary(response map[string]any, now time.Time) imageSummary {
	summary := imageSummary{
		Model:     stringField(response, "model"),
		Images:    extractImages(response),
		UpdatedAt: now.Format(time.RFC3339),
	}
	if usage, ok := response["usage"].(map[string]any); ok {
		summary.GeneratedImages = intField(usage, "generated_images")
		summary.TotalTokens = intField(usage, "total_tokens")
	}
	if summary.GeneratedImages == 0 {
		summary.GeneratedImages = len(summary.Images)
	}
	return summary
}

func extractImages(response map[string]any) []imageResult {
	data, ok := response["data"].([]any)
	if !ok {
		return nil
	}
	images := make([]imageResult, 0, len(data))
	for _, item := range data {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		images = append(images, imageResult{
			URL:  stringField(m, "url"),
			Size: stringField(m, "size"),
		})
	}
	return images
}

func stringField(m map[string]any, key string) string {
	value, _ := m[key].(string)
	return value
}

func intField(m map[string]any, key string) int {
	switch value := m[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}

func imageFileExt(rawURL, contentType string) string {
	switch {
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		return ".jpeg"
	}
	u, err := url.Parse(rawURL)
	if err == nil {
		ext := strings.ToLower(filepath.Ext(u.Path))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
			return ext
		}
	}
	return ".jpeg"
}

func responsePayload(response map[string]any) map[string]any {
	if data, ok := response["data"].(map[string]any); ok && data != nil {
		return data
	}
	return response
}

func chooseResponseError(root, payload map[string]any) any {
	if errValue, ok := payload["error"]; ok && errValue != nil {
		return errValue
	}
	if errValue, ok := root["error"]; ok && errValue != nil {
		return errValue
	}
	return nil
}

func extractString(value any, path ...string) string {
	current := value
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[key]
	}
	s, _ := current.(string)
	return s
}

func writeJSONFile(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败：%w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("写入文件失败 %s：%w", path, err)
	}
	return nil
}

func (a *App) getenv(key string) string {
	if a.Getenv == nil {
		return os.Getenv(key)
	}
	return a.Getenv(key)
}

func (a *App) now() time.Time {
	if a.Now == nil {
		return time.Now()
	}
	return a.Now()
}

func (a *App) makeTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%032x", a.now().UnixNano())
}

func (a *App) printRootUsage() {
	fmt.Fprintln(a.Err, "用法：autocom zlhub <create|get|image> [options]")
}

func (a *App) printZLHubUsage() {
	fmt.Fprintln(a.Err, "用法：autocom zlhub create [options]")
	fmt.Fprintln(a.Err, "用法：autocom zlhub get --id <task_id> [options]")
	fmt.Fprintln(a.Err, "用法：autocom zlhub image [options]")
}
