package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type input struct {
	TaskID string `json:"task_id"`
	Prompt string `json:"prompt"`
}

type output struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		write(output{Status: "failed", Error: fmt.Sprintf("read input: %v", err)})
		os.Exit(1)
	}

	var in input
	if len(data) > 0 {
		if err := json.Unmarshal(data, &in); err != nil {
			write(output{Status: "failed", Error: fmt.Sprintf("invalid json input: %v", err)})
			os.Exit(1)
		}
	}

	write(output{
		TaskID: in.TaskID,
		Status: "completed",
		Result: fmt.Sprintf("echo workflow received prompt: %s", in.Prompt),
	})
}

func write(out output) {
	_ = json.NewEncoder(os.Stdout).Encode(out)
}
