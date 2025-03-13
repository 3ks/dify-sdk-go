package dify

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RunStreamChatFlowWithHandler 方法
func (api *API) RunStreamChatFlowWithHandler(ctx context.Context, request ChatMessageRequest, handler EventHandler) error {
	request.ResponseMode = "streaming"
	req, err := api.createBaseRequest(ctx, http.MethodPost, "/v1/chat-messages", request)
	if err != nil {
		return err
	}

	resp, err := api.c.sendRequest(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %s: %s", resp.Status, readResponseBody(resp.Body))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading streaming response: %w", err)
		}

		if len(line) > 6 && string(line[:6]) == "data: " {
			var event struct {
				Event string `json:"event"`
			}
			if err := json.Unmarshal(line[6:], &event); err != nil {
				fmt.Println("Error decoding event type:", err)
				continue
			}

			switch event.Event {
			case EventTTSMessage, EventTTSMessageEnd:
				var ttsMsg TTSMessage
				if err := json.Unmarshal(line[6:], &ttsMsg); err != nil {
					fmt.Println("Error decoding TTS message:", err)
					continue
				}
				handler.HandleTTSMessage(ttsMsg)
			default:
				var streamResp StreamingResponse
				if err := json.Unmarshal(line[6:], &streamResp); err != nil {
					fmt.Println("Error decoding streaming response:", err)
					continue
				}
				handler.HandleStreamingResponse(streamResp)
				if event.Event == EventMessageEnd {
					return nil
				}
			}
		}
	}

	return nil
}
