<!-- // package main

// import (
// 	"context"
// 	"fmt"
// 	"os"

// 	openai "github.com/sashabaranov/go-openai"
// )

// func main() {
// 	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

// 	// Step 1: Upload your training file
// 	file, err := os.Open("training.jsonl")
// 	if err != nil {
// 		panic(err)
// 	}
// 	upload, err := client.CreateFile(context.Background(), openai.FileRequest{
// 		Purpose: "fine-tune",
// 		File:    file,
// 	})
// 	fmt.Println("Uploaded file ID:", upload.ID)

// 	// Step 2: Create fine-tuning job
// 	job, err := client.CreateFineTuningJob(context.Background(), openai.FineTuningJobRequest{
// 		TrainingFile: upload.ID,
// 		Model:        "gpt-4o-mini", // base model
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Println("Fine-tuning job started:", job.ID)
// } -->
