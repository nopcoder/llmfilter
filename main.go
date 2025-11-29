package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"
)

// Client handles communication with Ollama API
type client struct {
	baseURL    string
	httpClient *http.Client
}

// newClient creates a new Ollama client
func newClient(baseURL string) *client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// generateRequest represents a request to Ollama API
type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// generateResponse represents a response from Ollama API
type generateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

// generate sends a prompt to Ollama and returns the response
func (c *client) generate(model, prompt string) (string, error) {
	reqBody := generateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama (is it running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var generateResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&generateResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return generateResp.Response, nil
}

// evaluateContent evaluates content using Ollama with a question
func (c *client) evaluateContent(model string, question string, content string) (bool, error) {
	// Build prompt
	prompt := buildPrompt(question, content)

	// Get response from Ollama
	response, err := c.generate(model, prompt)
	if err != nil {
		return false, err
	}

	// Parse yes/no response
	return parseYesNoResponse(response), nil
}

// buildPrompt creates a prompt for evaluating content
func buildPrompt(question string, content string) string {
	var sb strings.Builder
	sb.WriteString("Question: ")
	sb.WriteString(question)
	sb.WriteString("\n\n")

	sb.WriteString("Content:\n")
	sb.WriteString(content)

	sb.WriteString("\n\nAnswer with only 'yes' or 'no':")
	return sb.String()
}

// parseYesNoResponse parses a yes/no response from Ollama
func parseYesNoResponse(response string) bool {
	response = strings.ToLower(strings.TrimSpace(response))

	// Check for positive responses
	positivePatterns := []string{"yes", "y", "true", "1", "correct", "affirmative"}
	if slices.Contains(positivePatterns, response) {
		return true
	}

	// Check for negative responses
	negativePatterns := []string{"no", "n", "false", "0", "incorrect", "negative"}
	if slices.Contains(negativePatterns, response) {
		return false
	}

	// Default to false if unclear
	return false
}

// evaluator handles item evaluation using Ollama
type evaluator struct {
	client *client
	model  string
}

// newEvaluator creates a new evaluator
func newEvaluator(baseURL, model string) *evaluator {
	return &evaluator{
		client: newClient(baseURL),
		model:  model,
	}
}

// evaluateContent evaluates content with a question
func (e *evaluator) evaluateContent(question string, content string) (bool, error) {
	return e.client.evaluateContent(e.model, question, content)
}

func main() {
	var (
		model     = flag.String("model", "llama3.1:latest", "Ollama model name")
		ollamaURL = flag.String("ollama-url", "http://localhost:11434", "Ollama API URL")
		question  = flag.String("question", "", "Question to ask the LLM for each line")
		keepIf    = flag.String("keep-if", "yes", "Keep lines where answer is 'yes' or 'no'")
		showAll   = flag.Bool("show-all", false, "Print all lines with +/- keep indicator")
		input     = flag.String("input", "", "Input file path (default: stdin)")
		output    = flag.String("output", "", "Output file path (default: stdout)")
	)
	flag.Parse()

	if *question == "" {
		fmt.Fprintf(os.Stderr, "Error: --question flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if *keepIf != "yes" && *keepIf != "no" {
		fmt.Fprintf(os.Stderr, "Error: --keep-if must be 'yes' or 'no'\n")
		os.Exit(1)
	}

	// Setup input
	var inputReader io.Reader = os.Stdin
	if *input != "" {
		file, err := os.Open(*input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		inputReader = file
	}

	// Setup output
	var outputWriter io.Writer = os.Stdout
	if *output != "" {
		file, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		outputWriter = file
	}

	// Create evaluator
	evaluator := newEvaluator(*ollamaURL, *model)

	// Read from input line by line
	scanner := bufio.NewScanner(inputReader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Evaluate the line
		answer, err := evaluator.evaluateContent(*question, line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error evaluating line: %v\n", err)
			continue
		}

		shouldKeep := (*keepIf == "yes" && answer) || (*keepIf == "no" && !answer)

		if *showAll {
			response := '-'
			if shouldKeep {
				response = '+'
			}
			fmt.Fprintf(outputWriter, "%c%s\n", response, line)
		} else if shouldKeep {
			fmt.Fprintf(outputWriter, "%s\n", line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
