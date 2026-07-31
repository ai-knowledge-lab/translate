package translate

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/ai-knowledge-lab/utils"
	"github.com/ai-knowledge-lab/utils/log"
	"github.com/ai-knowledge-lab/utils/types/ollama"
	"github.com/google/uuid"
)

type TranslateRequest struct {
	Text  string `json:"text" binding:"required"`
	From  string `json:"from" binding:"required"`
	To    string `json:"to" binding:"required"`
	Model string `json:"model" binding:"required"`
}

type TranslateResponse struct {
	TranslateID string `json:"translate_id"`
	Answer      string `json:"answer"`
	Think       string `json:"think"`
	Status      int    `json:"status"`
	Data        any    `json:"data"`
}

func (service *Service) Translate(ctx context.Context, text, from, to, model string) (chan TranslateResponse, error) {
	var response = make(chan TranslateResponse, 5000)

	// 1. 构造请求体
	reqBody := ollama.ChatRequest{
		Model: model,
		Messages: []ollama.ChatRequestMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf("请将以下%s文本翻译成%s，保持专业且自然的语气：\n\n%s", from, to, text),
			},
		},
		Stream: true, // 开启流式
		Options: map[string]any{
			"num_ctx": 16384,
		},
	}

	var result ollama.Response
	var isThinking = false
	var TranslateID = uuid.NewString()

	// 2. 将网络请求与流式解析全部放入后台协程，防止阻塞主协程，且安全管理流生命周期
	go func() {
		reqCtx, cancel := context.WithTimeout(context.WithValue(context.Background(), "x-request-id", ctx.Value("x-request-id")), duration())
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(debug.Stack())
				log.SugarContext(reqCtx).Errorw("流式翻译发生 Panic 异常", "error", err)
			}
			close(response)
			cancel()
		}()

		// 发起 HTTP 流式 POST 请求
		resp, err := client.R().
			SetBody(reqBody).
			SetDoNotParseResponse(true).
			Post("http://127.0.0.1:11434/api/chat")

		if err != nil {
			log.SugarContext(reqCtx).Errorf("❌ 连线大模型失败: %v", err)
			response <- TranslateResponse{
				TranslateID: TranslateID,
				Status:      500,
				Answer:      "连接翻译服务失败: " + err.Error(),
			}
			return
		}

		bodyStream := resp.RawBody()
		// 🌟 【关键修复 1】：将 defer Close() 放在异步协程中，保证整个流读取完毕后再关闭网络流
		defer bodyStream.Close()

		if resp.IsError() {
			log.SugarContext(reqCtx).Errorf("❌ 大模型服务返回错误状态码: %d", resp.StatusCode())
			response <- TranslateResponse{
				TranslateID: TranslateID,
				Status:      resp.StatusCode(),
				Answer:      fmt.Sprintf("翻译服务响应异常(状态码: %d)", resp.StatusCode()),
			}
			return
		}

		scanner := bufio.NewScanner(bodyStream)
		// 🌟 【关键修复 3】：扩充 Scanner 缓冲区容量至 1MB，避免大模型返回单行超大 JSON 导致截断报错
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// 解析当前行的碎片数据
			var chunk ollama.Response
			if err := json.Unmarshal(line, &chunk); err != nil {
				log.SugarContext(reqCtx).Errorf("解析流式片段失败: %v, 内容: %s", err, string(line))
				continue
			}

			// 思考链推送
			if chunk.Message.Thinking != "" {
				if !isThinking {
					isThinking = true
				}

				response <- TranslateResponse{
					TranslateID: TranslateID,
					Think:       chunk.Message.Thinking,
					Answer:      "",
					Status:      0,
					Data:        map[string]any{},
				}
				result.Message.Thinking += chunk.Message.Thinking
			}

			// 最终回复内容推送
			if chunk.Message.Content != "" {
				if isThinking {
					isThinking = false
				}
				response <- TranslateResponse{
					TranslateID: TranslateID,
					Think:       "",
					Answer:      chunk.Message.Content,
					Status:      0,
					Data:        map[string]any{},
				}
				result.Message.Content += chunk.Message.Content
			}

			result.Message.Role = chunk.Message.Role
			// 静态/末尾字段保存
			result.Model = chunk.Model
			result.CreatedAt = chunk.CreatedAt
			result.Done = chunk.Done
			result.DoneReason = chunk.DoneReason

			if chunk.TotalDuration > 0 {
				result.TotalDuration = chunk.TotalDuration
				result.LoadDuration = chunk.LoadDuration
				result.PromptEvalCount = chunk.PromptEvalCount
				result.PromptEvalDuration = chunk.PromptEvalDuration
				result.EvalCount = chunk.EvalCount
				result.EvalDuration = chunk.EvalDuration
			}

			if chunk.Done {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			log.SugarContext(reqCtx).Errorf("读取大模型流式响应出错: %v", err)
		}

		utils.WriteFile(fmt.Sprintf("translate-result-%s-sse.json", TranslateID), result)
	}()

	return response, nil
}
