package statistics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const captureLimit = 256 << 10

var defaultMu sync.RWMutex
var defaultService *Service

func SetDefault(service *Service) {
	defaultMu.Lock()
	defaultService = service
	defaultMu.Unlock()
}

func Default() *Service {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultService
}

type tracker struct {
	mu      sync.Mutex
	event   Event
	started time.Time
}

const trackerKey = "simple-one-api.statistics.tracker"

func SetRoute(c *gin.Context, clientModel, providerID, providerName, upstreamModel string) {
	value, ok := c.Get(trackerKey)
	if !ok {
		return
	}
	t := value.(*tracker)
	t.mu.Lock()
	t.event.ClientModel = clientModel
	t.event.ProviderID = providerID
	t.event.ProviderName = providerName
	t.event.UpstreamModel = upstreamModel
	t.mu.Unlock()
}

// SetUsage lets adapters report authoritative upstream usage before the
// response is serialized. A reported value takes precedence over body parsing.
func SetUsage(c *gin.Context, usage Usage) {
	value, ok := c.Get(trackerKey)
	if !ok {
		return
	}
	usage.Source = "upstream"
	t := value.(*tracker)
	t.mu.Lock()
	t.event.Usage = mergeUsage(t.event.Usage, usage)
	t.mu.Unlock()
}

// MarkFirstToken records the first meaningful streamed content delta.
func MarkFirstToken(c *gin.Context) {
	value, ok := c.Get(trackerKey)
	if !ok {
		return
	}
	t := value.(*tracker)
	t.mu.Lock()
	if t.event.FirstTokenMS == nil {
		elapsed := time.Since(t.started).Milliseconds()
		t.event.FirstTokenMS = &elapsed
	}
	t.mu.Unlock()
}

// ShareTracker lets protocol adapters route through an internal Gin context
// while still enriching the event owned by the original request.
func ShareTracker(destination, source *gin.Context) {
	if value, ok := source.Get(trackerKey); ok {
		destination.Set(trackerKey, value)
	}
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isTrackedPath(c.Request.URL.Path, c.Request.Method) {
			c.Next()
			return
		}
		requestID := uuid.NewString()
		c.Header("X-Request-ID", requestID)
		service := Default()
		if service == nil || !service.Settings().Enabled {
			c.Next()
			return
		}
		started := time.Now()
		event := Event{RequestID: requestID, CreatedAt: started.UTC(), Protocol: protocolForPath(c.Request.URL.Path), AccessKeyID: Fingerprint(accessToken(c))}
		t := &tracker{event: event, started: started}
		c.Set(trackerKey, t)
		capture := &captureWriter{ResponseWriter: c.Writer, started: started, context: c}
		c.Writer = capture
		c.Next()
		fallbackUsage := parseUsage(capture.Bytes())
		t.mu.Lock()
		if t.event.Usage.Source != "upstream" && fallbackUsage.Source == "upstream" {
			t.event.Usage = fallbackUsage
		}
		event = t.event
		t.mu.Unlock()
		event.StatusCode = c.Writer.Status()
		if event.StatusCode == 0 {
			event.StatusCode = http.StatusOK
		}
		event.LatencyMS = time.Since(started).Milliseconds()
		if !capture.firstWrite.IsZero() {
			value := capture.firstWrite.Sub(started).Milliseconds()
			event.FirstByteMS = &value
		}
		service.Record(event)
	}
}

func isTrackedPath(path, method string) bool {
	if method != http.MethodPost {
		return false
	}
	return strings.HasPrefix(path, "/v1/") || path == "/translate" || path == "/v2/translate"
}

func protocolForPath(path string) string {
	switch {
	case strings.HasSuffix(path, "/responses"):
		return "responses"
	case strings.HasSuffix(path, "/messages"):
		return "anthropic"
	case strings.HasSuffix(path, "/embeddings"):
		return "embeddings"
	case strings.Contains(path, "translate"):
		return "translation"
	default:
		return "chat"
	}
}

func accessToken(c *gin.Context) string {
	if token := strings.TrimSpace(c.GetHeader("x-api-key")); token != "" {
		return token
	}
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(authorization) > 7 && strings.EqualFold(authorization[:7], "Bearer ") {
		return strings.TrimSpace(authorization[7:])
	}
	return ""
}

func Fingerprint(key string) string {
	if key == "" {
		return "anonymous"
	}
	sum := sha256.Sum256([]byte(key))
	return "key-" + hex.EncodeToString(sum[:])[:12]
}

type captureWriter struct {
	gin.ResponseWriter
	started    time.Time
	firstWrite time.Time
	tail       []byte
	context    *gin.Context
	stream     strings.Builder
}

func (w *captureWriter) Write(data []byte) (int, error) {
	if w.firstWrite.IsZero() {
		w.firstWrite = time.Now()
	}
	w.append(data)
	return w.ResponseWriter.Write(data)
}

func (w *captureWriter) WriteString(data string) (int, error) {
	if w.firstWrite.IsZero() {
		w.firstWrite = time.Now()
	}
	w.append([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *captureWriter) append(data []byte) {
	w.observe(data)
	if len(data) >= captureLimit {
		w.tail = append(w.tail[:0], data[len(data)-captureLimit:]...)
		return
	}
	if overflow := len(w.tail) + len(data) - captureLimit; overflow > 0 {
		copy(w.tail, w.tail[overflow:])
		w.tail = w.tail[:len(w.tail)-overflow]
	}
	w.tail = append(w.tail, data...)
}

func (w *captureWriter) observe(data []byte) {
	var completeRoot map[string]any
	if json.Unmarshal(data, &completeRoot) == nil {
		if usageMap := findUsage(completeRoot); usageMap != nil && !ignoreUsageEvent(completeRoot) {
			SetUsage(w.context, usageFromMap(usageMap))
		}
	}
	w.stream.Write(data)
	content := w.stream.String()
	lastNewline := strings.LastIndexByte(content, '\n')
	if lastNewline < 0 {
		if w.stream.Len() > captureLimit {
			w.stream.Reset()
		}
		return
	}
	complete := content[:lastNewline+1]
	remainder := content[lastNewline+1:]
	w.stream.Reset()
	w.stream.WriteString(remainder)
	for _, line := range strings.Split(complete, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var root map[string]any
		if json.Unmarshal([]byte(payload), &root) != nil {
			continue
		}
		if usageMap := findUsage(root); usageMap != nil && !ignoreUsageEvent(root) {
			SetUsage(w.context, usageFromMap(usageMap))
		}
		if hasTokenDelta(root) {
			MarkFirstToken(w.context)
		}
	}
}

func (w *captureWriter) Bytes() []byte { return w.tail }

func parseUsage(data []byte) Usage {
	usage := Usage{Source: "missing"}
	candidates := [][]byte{data}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload != "" && payload != "[DONE]" {
				candidates = append(candidates, []byte(payload))
			}
		}
	}
	for _, candidate := range candidates {
		var root map[string]any
		if json.Unmarshal(candidate, &root) != nil {
			continue
		}
		usageMap := findUsage(root)
		if usageMap == nil {
			continue
		}
		if ignoreUsageEvent(root) {
			continue
		}
		usage = mergeUsage(usage, usageFromMap(usageMap))
	}
	if usage.TotalTokens == nil && usage.InputTokens != nil && usage.OutputTokens != nil {
		total := *usage.InputTokens + *usage.OutputTokens
		usage.TotalTokens = &total
	}
	return usage
}

func usageFromMap(usageMap map[string]any) Usage {
	usage := Usage{
		InputTokens:      number(usageMap, "prompt_tokens", "input_tokens"),
		OutputTokens:     number(usageMap, "completion_tokens", "output_tokens"),
		TotalTokens:      number(usageMap, "total_tokens"),
		CachedTokens:     number(usageMap, "cached_tokens", "cache_read_input_tokens"),
		CacheWriteTokens: number(usageMap, "cache_creation_input_tokens"),
		ReasoningTokens:  number(usageMap, "reasoning_tokens"),
		Source:           "upstream",
	}
	if details, ok := usageMap["input_tokens_details"].(map[string]any); ok {
		if v := number(details, "cached_tokens"); v != nil {
			usage.CachedTokens = v
		}
	}
	if details, ok := usageMap["prompt_tokens_details"].(map[string]any); ok {
		if v := number(details, "cached_tokens"); v != nil {
			usage.CachedTokens = v
		}
	}
	if details, ok := usageMap["output_tokens_details"].(map[string]any); ok {
		if v := number(details, "reasoning_tokens"); v != nil {
			usage.ReasoningTokens = v
		}
	}
	if details, ok := usageMap["completion_tokens_details"].(map[string]any); ok {
		if v := number(details, "reasoning_tokens"); v != nil {
			usage.ReasoningTokens = v
		}
	}
	return usage
}

func mergeUsage(current, incoming Usage) Usage {
	if incoming.Source != "upstream" {
		return current
	}
	current.Source = "upstream"
	for target, source := range map[**int64]*int64{
		&current.InputTokens: incoming.InputTokens, &current.OutputTokens: incoming.OutputTokens,
		&current.CachedTokens: incoming.CachedTokens, &current.CacheWriteTokens: incoming.CacheWriteTokens,
		&current.ReasoningTokens: incoming.ReasoningTokens, &current.TotalTokens: incoming.TotalTokens,
	} {
		if source != nil {
			*target = source
		}
	}
	if incoming.TotalTokens == nil && (incoming.InputTokens != nil || incoming.OutputTokens != nil) && current.InputTokens != nil && current.OutputTokens != nil {
		total := *current.InputTokens + *current.OutputTokens
		current.TotalTokens = &total
	}
	return current
}

func ignoreUsageEvent(root map[string]any) bool {
	eventType, _ := root["type"].(string)
	if eventType != "message_start" {
		return false
	}
	usage := findUsage(root)
	for _, name := range []string{"prompt_tokens", "completion_tokens", "input_tokens", "output_tokens", "total_tokens"} {
		if value := number(usage, name); value != nil && *value != 0 {
			return false
		}
	}
	return true
}

func hasTokenDelta(root map[string]any) bool {
	if eventType, _ := root["type"].(string); strings.HasSuffix(eventType, ".delta") {
		if text, ok := root["delta"].(string); ok && text != "" {
			return true
		}
		if delta, ok := root["delta"].(map[string]any); ok {
			for _, name := range []string{"text", "partial_json"} {
				if text, ok := delta[name].(string); ok && text != "" {
					return true
				}
			}
		}
	}
	choices, _ := root["choices"].([]any)
	for _, value := range choices {
		choice, _ := value.(map[string]any)
		delta, _ := choice["delta"].(map[string]any)
		if content, ok := delta["content"].(string); ok && content != "" {
			return true
		}
		if calls, ok := delta["tool_calls"].([]any); ok && len(calls) > 0 {
			return true
		}
	}
	return false
}

func findUsage(root map[string]any) map[string]any {
	if value, ok := root["usage"].(map[string]any); ok {
		return value
	}
	if message, ok := root["message"].(map[string]any); ok {
		if value, ok := message["usage"].(map[string]any); ok {
			return value
		}
	}
	if response, ok := root["response"].(map[string]any); ok {
		if value, ok := response["usage"].(map[string]any); ok {
			return value
		}
	}
	return nil
}

func number(values map[string]any, names ...string) *int64 {
	for _, name := range names {
		if raw, ok := values[name].(float64); ok {
			value := int64(raw)
			return &value
		}
	}
	return nil
}
