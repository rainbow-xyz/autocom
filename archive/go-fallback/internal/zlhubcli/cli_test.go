package zlhubcli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildCreatePayloadFromFlags(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ZLHub.CallbackURL = "https://callback.example.com/zlhub"

	opts := createOptions{
		prompt: "参考图片1生成视频",
		images: stringList{
			"reference_image=https://example.com/a.jpg",
			"first_frame=https://example.com/first.jpg",
		},
		videos:        stringList{"reference_video=https://example.com/ref.mp4"},
		audios:        stringList{"reference_audio=https://example.com/bgm.mp3"},
		ratio:         "16:9",
		resolution:    "480p",
		duration:      11,
		generateAudio: false,
		watermark:     false,
	}
	payload, err := buildCreatePayload(opts, cfg, map[string]bool{
		"ratio":          true,
		"resolution":     true,
		"duration":       true,
		"generate-audio": true,
		"watermark":      true,
	})
	if err != nil {
		t.Fatalf("buildCreatePayload returned error: %v", err)
	}
	if payload["callback_url"] != cfg.ZLHub.CallbackURL {
		t.Fatalf("callback_url = %v", payload["callback_url"])
	}
	if payload["ratio"] != "16:9" {
		t.Fatalf("ratio = %v", payload["ratio"])
	}
	if payload["duration"] != 11 {
		t.Fatalf("duration = %v", payload["duration"])
	}
	if payload["resolution"] != "480p" {
		t.Fatalf("resolution = %v", payload["resolution"])
	}
	if payload["generate_audio"] != false {
		t.Fatalf("generate_audio = %v", payload["generate_audio"])
	}
	content, ok := payload["content"].([]any)
	if !ok {
		t.Fatalf("content type = %T", payload["content"])
	}
	if len(content) != 5 {
		t.Fatalf("content length = %d", len(content))
	}
}

func TestBuildCreatePayloadFromFileWithOverrides(t *testing.T) {
	dir := t.TempDir()
	requestFile := filepath.Join(dir, "request.json")
	if err := os.WriteFile(requestFile, []byte(`{"model":"from-file","content":[{"type":"text","text":"hello"}],"ratio":"1:1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.ZLHub.CallbackURL = "https://config.example.com/callback"
	opts := createOptions{
		file:        requestFile,
		callbackURL: "https://cli.example.com/callback",
		duration:    6,
	}
	payload, err := buildCreatePayload(opts, cfg, map[string]bool{
		"callback-url": true,
		"duration":     true,
	})
	if err != nil {
		t.Fatalf("buildCreatePayload returned error: %v", err)
	}
	if payload["model"] != "from-file" {
		t.Fatalf("model = %v", payload["model"])
	}
	if payload["ratio"] != "1:1" {
		t.Fatalf("ratio = %v", payload["ratio"])
	}
	if payload["duration"] != 6 {
		t.Fatalf("duration = %v", payload["duration"])
	}
	if payload["callback_url"] != "https://cli.example.com/callback" {
		t.Fatalf("callback_url = %v", payload["callback_url"])
	}
}

func TestBuildImagePayloadFromFlags(t *testing.T) {
	cfg := DefaultConfig()
	opts := imageOptions{
		prompt:         "生成一张物理竞赛海报",
		model:          "doubao-seedream-5.0-lite",
		size:           "1440x2560",
		responseFormat: "url",
		watermark:      false,
	}
	payload, err := buildImagePayload(opts, cfg, map[string]bool{
		"model":           true,
		"size":            true,
		"response-format": true,
		"watermark":       true,
	})
	if err != nil {
		t.Fatalf("buildImagePayload returned error: %v", err)
	}
	if payload["prompt"] != "生成一张物理竞赛海报" {
		t.Fatalf("prompt = %v", payload["prompt"])
	}
	if payload["model"] != "doubao-seedream-5.0-lite" {
		t.Fatalf("model = %v", payload["model"])
	}
	if payload["size"] != "1440x2560" {
		t.Fatalf("size = %v", payload["size"])
	}
	if payload["response_format"] != "url" {
		t.Fatalf("response_format = %v", payload["response_format"])
	}
	if payload["watermark"] != false {
		t.Fatalf("watermark = %v", payload["watermark"])
	}
}

func TestCreateSavesFiles(t *testing.T) {
	var gotAuth string
	var gotPath string
	app := testApp()
	app.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		return jsonResponse(http.StatusOK, `{"id":"cgt-test","status":"running"}`), nil
	})}

	outDir := t.TempDir()
	var stdout bytes.Buffer
	app.Out = &stdout
	err := app.Run([]string{
		"zlhub", "create",
		"--api-base", "https://zlhub.test",
		"--prompt", "参考图片1生成视频",
		"--image", "reference_image=https://example.com/a.jpg",
		"--out-dir", outDir,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotPath != "/v1/task/create" {
		t.Fatalf("path = %q", gotPath)
	}
	assertJSONFileHas(t, filepath.Join(outDir, "zlhub", "request.json"), "content")
	var summary taskSummary
	readJSONFile(t, filepath.Join(outDir, "zlhub", "task.json"), &summary)
	if summary.TaskID != "cgt-test" || summary.Status != "running" {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestImageSavesResponseAndDownloadsFile(t *testing.T) {
	var gotAuth string
	var gotPostPath string
	var gotImagePath string
	app := testApp()
	app.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost {
			gotAuth = r.Header.Get("Authorization")
			gotPostPath = r.URL.Path
			return jsonResponse(http.StatusOK, `{"model":"doubao-seedream-5-0-260128","data":[{"url":"https://asset.test/image.jpeg","size":"1440x2560"}],"usage":{"generated_images":1,"total_tokens":14400}}`), nil
		}
		gotImagePath = r.URL.Path
		resp := jsonResponse(http.StatusOK, "fake image")
		resp.Header.Set("Content-Type", "image/jpeg")
		return resp, nil
	})}

	outDir := t.TempDir()
	err := app.Run([]string{
		"zlhub", "image",
		"--api-base", "https://zlhub.test",
		"--prompt", "生成一张物理竞赛海报",
		"--out-dir", outDir,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotPostPath != "/v1/images/generations" {
		t.Fatalf("post path = %q", gotPostPath)
	}
	if gotImagePath != "/image.jpeg" {
		t.Fatalf("image path = %q", gotImagePath)
	}
	assertJSONFileHas(t, filepath.Join(outDir, "zlhub", "image", "request.json"), "prompt")
	var summary imageSummary
	readJSONFile(t, filepath.Join(outDir, "zlhub", "image", "summary.json"), &summary)
	if summary.GeneratedImages != 1 || summary.TotalTokens != 14400 {
		t.Fatalf("summary = %+v", summary)
	}
	if _, err := os.Stat(filepath.Join(outDir, "zlhub", "image", "image_1.jpeg")); err != nil {
		t.Fatalf("downloaded image missing: %v", err)
	}
}

func TestGetSavesStatuses(t *testing.T) {
	cases := []struct {
		name     string
		response string
		wantURL  string
		wantErr  any
	}{
		{
			name:     "running",
			response: `{"id":"cgt-test","status":"running"}`,
		},
		{
			name:     "succeeded",
			response: `{"id":"cgt-test","status":"succeeded","content":{"video_url":"https://example.com/final.mp4"}}`,
			wantURL:  "https://example.com/final.mp4",
		},
		{
			name:     "succeeded",
			response: `{"code":"success","data":{"id":"cgt-test","status":"succeeded","content":{"video_url":"https://example.com/wrapped.mp4"}}}`,
			wantURL:  "https://example.com/wrapped.mp4",
		},
		{
			name:     "failed",
			response: `{"id":"cgt-test","status":"failed","error":{"message":"生成失败"}}`,
			wantErr:  map[string]any{"message": "生成失败"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outDir := t.TempDir()
			app := testApp()
			app.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, tc.response), nil
			})}
			err := app.Run([]string{
				"zlhub", "get",
				"--api-base", "https://zlhub.test",
				"--id", "cgt-test",
				"--out-dir", outDir,
			})
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			var summary taskSummary
			readJSONFile(t, filepath.Join(outDir, "zlhub", "task.json"), &summary)
			if summary.Status != tc.name {
				t.Fatalf("status = %q", summary.Status)
			}
			if summary.VideoURL != tc.wantURL {
				t.Fatalf("video_url = %q", summary.VideoURL)
			}
			if tc.wantErr != nil && summary.Error == nil {
				t.Fatalf("error should not be nil")
			}
		})
	}
}

func TestMissingAPIKeyReturnsChineseError(t *testing.T) {
	app := testApp()
	app.Getenv = func(string) string { return "" }
	err := app.Run([]string{
		"zlhub", "create",
		"--prompt", "hello",
		"--out-dir", t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "缺少环境变量 ZLHUB_API_KEY，请先在宿主机环境变量中配置 API Key" {
		t.Fatalf("error = %q", got)
	}
}

func TestMakeTraceIDIs32Characters(t *testing.T) {
	app := testApp()
	traceID := app.makeTraceID()
	if len(traceID) != 32 {
		t.Fatalf("trace id length = %d, trace id = %q", len(traceID), traceID)
	}
}

func TestNon2xxResponseIsSaved(t *testing.T) {
	outDir := t.TempDir()
	app := testApp()
	app.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"error":"bad request"}`), nil
	})}
	err := app.Run([]string{
		"zlhub", "create",
		"--api-base", "https://zlhub.test",
		"--prompt", "hello",
		"--out-dir", outDir,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertJSONFileHas(t, filepath.Join(outDir, "zlhub", "create_response.json"), "error")
}

func testApp() *App {
	return &App{
		Out:        &bytes.Buffer{},
		Err:        &bytes.Buffer{},
		Getenv:     func(key string) string { return map[string]string{"ZLHUB_API_KEY": "test-key"}[key] },
		HTTPClient: http.DefaultClient,
		Now: func() time.Time {
			return time.Date(2026, 4, 21, 15, 0, 0, 0, time.FixedZone("CST", 8*3600))
		},
	}
}

func assertJSONFileHas(t *testing.T, path string, key string) {
	t.Helper()
	var value map[string]any
	readJSONFile(t, path, &value)
	if _, ok := value[key]; !ok {
		t.Fatalf("%s missing key %s", path, key)
	}
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
