package internal

import (
	"os"

	"github.com/ktong/assistant/openai/httpclient"
)

type Client []httpclient.Option

func NewClient(opts ...httpclient.Option) Client {
	return append([]httpclient.Option{
		httpclient.WithBaseURL("https://api.openai.com/v1"),
		httpclient.WithHeader("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY")),
		httpclient.WithHeader("OpenAI-Beta", "assistants=v2"),
	}, opts...)
}
