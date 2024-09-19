// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package examples

import (
	"context"

	"github.com/ktong/assistant"
	"github.com/ktong/assistant/openai"
)

// Example_cot demonstrates the simplest usage of single text only assistant.
func Example_cot() {
	asst := assistant.Assistant{
		Name: "Chain of Thought",
		Instructions: `
You are an AI assistant designed to think through problems step-by-step using Chain-of-Thought (COT) prompting. Before providing any answer, you must:
- Understand the Problem: Carefully read and understand the user's question or request.
- Break Down the Reasoning Process: Outline the steps required to solve the problem or respond to the request logically and sequentially.Think aloud and describe each step in detail.
- Explain Each Step: Provide reasoning or calculations for each step, explaining how you arrive at each part of your answer.
- Arrive at the Final Answer: Only after completing all steps, provide the final answer or solution.
- Review the Thought Process: Double-check the reasoning for errors or gaps before finalizing your response.
Always aim to make your thought process transparent and logical, helping users understand how you reached your conclusion.
`,
	}

	assistant.SetDefaultExecutor(openai.NewExecutor())
	defer func() {
		if err := asst.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	var thread assistant.Thread
	if err := asst.Run(context.Background(), &thread, assistant.Message{Role: assistant.RoleUser, Content: []assistant.Content{assistant.Text{Text: `
If x is the average (arithmetic mean) of m and 9, y is the average of 2m and 15, and z is the average of 3m and 18,
what is the average of x, y, and z in terms of m?

A) m+6
B) m+7
C) 2m+14
D) 3m+21
`}}}); err != nil {
		panic(err)
	}

	println(thread.Messages[len(thread.Messages)-1].Content[0].(assistant.Text).Text)
	// Output:
}
