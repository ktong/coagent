package examples

import (
	"context"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/openai"
)

// Example_weather demonstrates the usage of single text assistant with [function calling].
//
// [function calling]: https://platform.openai.com/docs/assistants/tools/function-calling
func Example_weather() {
	asst := assistant.Assistant{
		Name:         "Weather Bot",
		Instructions: "You are a weather bot.",
		Tools: []assistant.Tool{
			assistant.FunctionFor[temperatureRequest, float32](getCurrentTemperature),
			assistant.FunctionFor[rainRequest, float32](getRainProbability),
		},
	}

	assistant.SetDefaultExecutor(openai.NewExecutor())
	defer func() {
		_ = asst.Shutdown(context.Background())
	}()

	var thread assistant.Thread
	if err := asst.Run(context.Background(), &thread, assistant.TextMessage(
		"What's the weather in San Francisco today and the likelihood it'll rain?",
	)); err != nil {
		panic(err)
	}

	println(thread.Messages[len(thread.Messages)-1].Content[0].(assistant.Text).Text)
	// Output:
}

type (
	rainRequest struct {
		Location string `json:"location" description:"The city and state, e.g., San Francisco, CA"`
	}
	temperatureRequest struct {
		Location string `json:"location" description:"The city and state, e.g., San Francisco, CA"`
		Unit     string `json:"unit" description:"The temperature unit to use. Infer this from the user's location." enum:"Celsius,Fahrenheit"`
	}
)

func getCurrentTemperature(context.Context, temperatureRequest) (float32, error) {
	return 72, nil
}

func getRainProbability(context.Context, rainRequest) (float32, error) {
	return 0.2, nil
}
