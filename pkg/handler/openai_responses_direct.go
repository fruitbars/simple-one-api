package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openaisdk "github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/statistics"
	"simple-one-api/pkg/utils"
)

// tryDirectResponses preserves the Responses protocol when the selected
// provider speaks Responses too. It returns handled=false for compatibility
// routes, allowing the existing Chat/Anthropic adapters to run unchanged.
func tryDirectResponses(c *gin.Context, body []byte, request responsesRequest) (handled bool, err error) {
	if strings.TrimSpace(request.Model) == "" {
		return false, nil
	}
	s, serviceModelName, lookupErr := getModelDetails(&openaisdk.ChatCompletionRequest{Model: request.Model})
	if lookupErr != nil || strings.ToLower(strings.TrimSpace(s.UpstreamProtocol)) != config.UpstreamProtocolResponses {
		return false, nil
	}
	clientModel := request.Model
	mappedModel := config.GetModelMapping(s, config.GetModelRedirect(s, serviceModelName))
	statistics.SetRoute(c, clientModel, s.ServiceID, s.ServiceName, mappedModel)
	tokenCost := estimateResponsesRequestTokens(body, request.MaxOutputTokens)
	candidates := mycommon.OrderCredentialCandidates(s, clientModel, tokenCost)
	if len(candidates) == 0 {
		return true, fmt.Errorf("no healthy credential is available")
	}
	requestTimeout := s.Timeout
	if requestTimeout <= 0 {
		requestTimeout = defaultReqTimeout
	}
	requestCtx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(requestTimeout)*time.Second)
	defer cancel()
	for index, candidate := range candidates {
		release, limitErr := acquireAttemptLimitsWithTokenCost(requestCtx, s, candidate, tokenCost, clientModel, serviceModelName, mappedModel)
		if limitErr != nil {
			if index == len(candidates)-1 {
				return true, limitErr
			}
			continue
		}
		var transport *http.Transport
		if config.IsProxyEnabled(s) {
			_, _, proxyTransport, proxyErr := config.GetConfProxyTransport()
			if proxyErr != nil {
				release()
				return true, proxyErr
			}
			transport = proxyTransport
		}
		mycommon.ReserveCredentialCapacity(candidate.Credentials, candidate.ID, clientModel, tokenCost)
		directParams := &directResponsesParams{ctx: requestCtx, modelDetails: s, creds: candidate.Credentials, clientModel: clientModel, mappedModel: mappedModel, body: body, stream: request.Stream, httpTransport: transport}
		err = sendDirectResponses(c, directParams)
		release()
		if err == nil {
			config.RecordProviderResult(candidate.ID, clientModel, true)
			return true, nil
		}
		config.RecordProviderResult(candidate.ID, clientModel, false)
		if isUpstreamRateLimit(err) {
			mycommon.MarkCredentialCooldown(candidate.ID, clientModel, 30*time.Second)
		}
		if !shouldRetryCredential(err) || c.Writer.Written() || index == len(candidates)-1 {
			return true, err
		}
	}
	return true, err
}

func estimateResponsesRequestTokens(body []byte, maxOutputTokens int) int {
	inputEstimate := (len(body) + 3) / 4
	if maxOutputTokens < 0 {
		maxOutputTokens = 0
	}
	if inputEstimate+maxOutputTokens < 1 {
		return 1
	}
	return inputEstimate + maxOutputTokens
}

type directResponsesParams struct {
	ctx           context.Context
	modelDetails  *config.ModelDetails
	creds         map[string]interface{}
	clientModel   string
	mappedModel   string
	body          []byte
	stream        bool
	httpTransport *http.Transport
}

func sendDirectResponses(c *gin.Context, params *directResponsesParams) error {
	payload := make(map[string]any)
	if err := json.Unmarshal(params.body, &payload); err != nil {
		return fmt.Errorf("invalid Responses request: %w", err)
	}
	payload["model"] = params.mappedModel
	if reasoning, ok := payload["reasoning"].(map[string]any); ok && reasoning["effort"] == "max" {
		reasoning["effort"] = "high"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	apiKey, _ := utils.GetStringFromMap(params.creds, config.KEYNAME_API_KEY)
	req, err := http.NewRequestWithContext(params.ctx, http.MethodPost, responsesEndpoint(params.modelDetails.ServerURL), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	if params.stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	client := utils.NewHTTPClient(params.httpTransport, responseTimeout(params.modelDetails.Timeout))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return utils.NewHTTPStatusError(resp.StatusCode, resp.Status, strings.TrimSpace(string(errorBody)))
	}
	for key, values := range resp.Header {
		if strings.EqualFold(key, "Content-Length") || strings.EqualFold(key, "Connection") {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	c.Status(resp.StatusCode)
	buffer := make([]byte, 32<<10)
	for {
		read, readErr := resp.Body.Read(buffer)
		if read > 0 {
			if _, writeErr := c.Writer.Write(buffer[:read]); writeErr != nil {
				return writeErr
			}
			if flusher, ok := c.Writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}
