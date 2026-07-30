package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ai-knowledge-lab/utils"
	"github.com/ai-knowledge-lab/utils/llm"
	"github.com/ai-knowledge-lab/utils/log"
	"github.com/ai-knowledge-lab/utils/types/ollama"
	"github.com/go-resty/resty/v2"
)

var (
	client = resty.New().
		SetTimeout(time.Hour*2).
		SetHeader("Content-Type", "application/json")
	model    = llm.Qwen38B
	endpoint = "http://127.0.0.1:11434/api/chat"
)

func main() {
	var filename = "text.txt"
	file, err := os.ReadFile(filename)
	if err != nil {
		log.Sugar().Fatalw("", "error", err)
	}

	// 构造请求体
	reqBody := ollama.ChatRequest{
		Model: model,
		Messages: []ollama.ChatRequestMessage{
			// {
			// 	Role:    "system",
			// 	Content: "你是一个严谨的计算机技术翻译机器人。\n【死命令】：\n1. 必须逐句翻译，不准遗漏任何段落。\n2. 遇到不懂的专业词汇（如 Kubernetes、Pod、goroutine），一律保持原样，禁止瞎翻。\n3. 永远只能输出翻译后的中文，禁止输出任何解释、禁止向用户提问、禁止夹杂英文说教。",
			// },
			{
				Role:    "user",
				Content: fmt.Sprintf("请将以下文本翻译成中文，保持专业且自然的语气：\n\n%s", file),
			},
		},
		Stream: true, // 开启流式
		Options: map[string]any{
			"num_ctx": 16384,
			// 	"temperature": 0.0, // 🌟 降温到 0！让模型极度死板、严谨，每次都选概率最高的词
			// 	"top_p":       0.1, // 🌟 缩窄采样范围，绝不瞎抖机灵
		},
	}

	fmt.Println("🚀 Resty 管道已就绪！正在发起翻译任务...")
	fmt.Printf("🤖 等待%s模型回应：", model)

	// 链式发起 POST 请求
	resp, err := client.R().
		SetBody(reqBody).
		SetDoNotParseResponse(true).
		Post(endpoint)

	if err != nil {
		log.Sugar().Fatalf("\n❌ 请求发起失败: %v", err)
	}

	// 核心：拿到原始的 Body 流，务必在结束时关闭
	bodyStream := resp.RawBody()
	defer bodyStream.Close()

	if resp.IsError() {
		errBytes, _ := io.ReadAll(bodyStream)
		log.Sugar().Fatalf("\n❌ 服务端返回错误，状态码: %d, 详情: %s", resp.StatusCode(), string(errBytes))
	}

	var result ollama.Response
	var isThinking = false

	scanner := bufio.NewScanner(bodyStream)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// 解析当前行的碎片数据
		var chunk ollama.Response
		if err := json.Unmarshal(line, &chunk); err != nil {
			fmt.Printf("\n❌ 解析流式片段失败: %v, 内容: %s\n", err, string(line))
			continue
		}

		// 动态字段一：追加思考链
		if chunk.Message.Thinking != "" {
			if !isThinking {
				fmt.Print("\n🤔🤔🤔 思考中: ")
				isThinking = true
			}
			fmt.Print(chunk.Message.Thinking)
			result.Message.Thinking += chunk.Message.Thinking
		}

		// 动态字段二：追加最终回复内容
		if chunk.Message.Content != "" {
			if isThinking {
				fmt.Print("\n\n💡💡💡 翻译结果: \n")
				isThinking = false
			}
			fmt.Print(chunk.Message.Content)
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
		fmt.Printf("\n❌ 读取流式响应出错: %v\n", err)
	}

	utils.WriteFile("result.json", result)
	fmt.Println("\n\n✅ 【翻译任务完成，结果已保存！】")
}
