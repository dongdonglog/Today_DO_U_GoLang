package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	question := flag.String("question", "", "customer question")
	flag.Parse()
	if strings.TrimSpace(*question) == "" || len(*question) > 4000 {
		fatal("question is required and must be at most 4000 bytes", 2)
	}
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	modelName := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if apiKey == "" || modelName == "" {
		fatal("OPENAI_API_KEY and OPENAI_MODEL are required", 2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  apiKey,
		Model:   modelName,
		BaseURL: strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
		Timeout: 55 * time.Second,
	})
	if err != nil {
		fatal("create chat model: "+err.Error(), 1)
	}

	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是企业客服知识助手。只提供谨慎的候选答复；信息不足时说明需要人工核实，不要承诺退款或执行订单操作。"),
		schema.UserMessage("{question}"),
	)
	chain := compose.NewChain[map[string]any, *schema.Message]().
		AppendChatTemplate(template).
		AppendChatModel(chatModel)
	runnable, err := chain.Compile(ctx)
	if err != nil {
		fatal("compile Eino chain: "+err.Error(), 1)
	}

	output, err := runnable.Invoke(ctx, map[string]any{"question": *question})
	if err != nil {
		if ctx.Err() != nil {
			fatal("Eino chain deadline exceeded", 1)
		}
		fatal("invoke Eino chain: "+err.Error(), 1)
	}
	if output == nil || strings.TrimSpace(output.Content) == "" {
		fatal("chain returned no text content", 1)
	}
	fmt.Println(output.Content)
}

func fatal(message string, code int) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(code)
}
