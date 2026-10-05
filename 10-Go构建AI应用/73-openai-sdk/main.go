package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func main() {
	question := flag.String("question", "", "question to send to the model")
	flag.Parse()
	if strings.TrimSpace(*question) == "" || len(*question) > 4000 {
		fmt.Fprintln(os.Stderr, "question is required and must be at most 4000 bytes")
		os.Exit(2)
	}
	if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY is required")
		os.Exit(2)
	}

	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "gpt-5.2"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := openai.NewClient()
	response, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model: openai.ChatModel(model),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(*question),
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "OpenAI request failed:", err)
		os.Exit(1)
	}
	if response == nil {
		fmt.Fprintln(os.Stderr, "OpenAI returned an empty response")
		os.Exit(1)
	}
	answer := strings.TrimSpace(response.OutputText())
	if answer == "" {
		fmt.Fprintln(os.Stderr, "response contains no plain-text output")
		os.Exit(1)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		fmt.Fprintln(os.Stderr, "request deadline exceeded")
		os.Exit(1)
	}
	fmt.Println(answer)
}
