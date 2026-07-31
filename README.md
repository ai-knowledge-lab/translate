# 本地 AI 翻译服务

这是一个基于 Go、Gin 和 Ollama 的本地翻译项目。服务通过本机 Ollama 调用大语言模型，并使用 SSE（Server-Sent Events）将思考过程和翻译结果流式返回到网页。

项目默认提供 Web 翻译页面，也可以直接调用 HTTP 接口进行翻译。文本和模型推理均在本机完成。

## 支持的模型

- `deepseek-r1:8b`
- `qwen3.5:9b`
- `qwen3:8b`

使用某个模型前，需要先通过 Ollama 将该模型下载到本机。

## 环境准备

请先安装：

- [Ollama](https://docs.ollama.com/quickstart)
- Go（版本应满足 `go.mod` 中的要求，当前为 Go 1.26.5）

## 启动服务

### 1. 启动 Ollama

确保 Ollama 正在运行：

```bash
ollama serve
```

Windows 或 macOS 上如果 Ollama 桌面程序已经启动，通常不需要再次执行此命令。Ollama API 默认监听 `http://127.0.0.1:11434`。

### 2. 下载翻译模型

按需下载一个或多个模型：

```bash
ollama pull deepseek-r1:8b
ollama pull qwen3.5:9b
ollama pull qwen3:8b
```

查看本机已有模型：

```bash
ollama ls
```

### 3. 启动翻译服务

在项目根目录执行：

```bash
go mod download
go run .
```

翻译服务启动后监听：

```text
http://127.0.0.1:8080
```

可以通过健康检查确认服务状态：

```bash
curl http://127.0.0.1:8080/ping
```

返回 `pong` 说明服务已正常启动。

## 使用网页进行翻译

1. 浏览器打开 `http://127.0.0.1:8080`。
2. 选择源语言和目标语言。
3. 选择已经下载到本机的模型。
4. 输入需要翻译的文本。
5. 点击“立即翻译”，页面会流式显示模型的思考过程和翻译结果。

翻译过程中可以点击停止按钮中断接收，也可以开启实时自动翻译。

## 调用翻译接口

接口地址：

```text
POST http://127.0.0.1:8080/translate
```

请求体必须包含以下字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `text` | string | 需要翻译的文本 |
| `from` | string | 源语言，例如 `英文` |
| `to` | string | 目标语言，例如 `中文` |
| `model` | string | Ollama 中已安装的模型名称 |

调用示例：

```bash
curl -N -X POST http://127.0.0.1:8080/translate \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "text": "Hello, welcome to the local translation service!",
    "from": "英文",
    "to": "中文",
    "model": "deepseek-r1:8b"
  }'
```

Windows PowerShell 中可将 `curl` 替换为 `curl.exe`。`-N` 用于关闭输出缓冲，以便实时查看 SSE 翻译内容。

## 服务路由

| 路由 | 方法 | 说明 |
| --- | --- | --- |
| `/` | GET | Web 翻译页面 |
| `/sse` | GET | SSE 翻译示例页面 |
| `/ping` | GET | 服务健康检查 |
| `/translate` | POST | 流式翻译接口 |

## 常见问题

### 无法连接翻译服务

确认 Ollama 已启动，并检查 `http://127.0.0.1:11434` 是否可访问。

### 提示模型不存在

模型名称必须与 `ollama ls` 中显示的名称一致。缺少模型时执行：

```bash
ollama pull <模型名称>
```

### 接口返回 400

检查请求体是否包含 `text`、`from`、`to` 和 `model` 四个必填字段，并确保请求头为 `Content-Type: application/json`。

### 翻译速度较慢

本地推理速度取决于模型大小和电脑的 CPU、GPU、内存配置。首次使用模型时还可能包含模型加载时间。

## 停止服务

在运行 `go run .` 的终端中按 `Ctrl+C`，服务会执行安全关闭。
