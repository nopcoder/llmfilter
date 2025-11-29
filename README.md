# llmfilter

A command-line tool that filters text lines using Large Language Models (LLMs) via Ollama.

## Description

llmfilter reads input line by line and uses an LLM to evaluate each line against a specified question. It filters the lines based on whether the LLM's response is "yes" or "no", allowing you to programmatically filter text content using natural language queries.

## Requirements

- Go 1.25.4 or later
- [Ollama](https://ollama.ai/) running locally with a compatible model

## Installation

```bash
go install github.com/nopcoder/llmfilter@latest
```

## Usage

```bash
llmfilter --question "Is this a programming language?" --input input.txt --output filtered.txt
```

### Command Line Flags

- `--model`: Ollama model name (default: "llama3.1:latest")
- `--ollama-url`: Ollama API URL (default: "http://localhost:11434")
- `--question`: Question to ask the LLM for each line (required)
- `--keep-if`: Keep lines where answer is 'yes' or 'no' (default: "yes")
- `--show-all`: Print all lines with +/- keep indicator
- `--input`: Input file path (default: stdin)
- `--output`: Output file path (default: stdout)

### Question from File

You can load the question from a file by prefixing the question with `@`:

```bash
llmfilter --question @question.txt --input data.txt
```

### Examples

Filter programming languages from a list:

```bash
echo -e "Python\nJava\nEnglish\nGo\nSpanish" | llmfilter --question "Is this a programming language?"
```

Output:
```
Python
Java
Go
```

Show all lines with indicators:

```bash
echo -e "Python\nJava\nEnglish\nGo\nSpanish" | llmfilter --question "Is this a programming language?" --show-all
```

Output:
```
+Python
+Java
-English
+Go
-Spanish
```

Keep lines where the answer is "no":

```bash
echo -e "Python\nJava\nEnglish\nGo\nSpanish" | llmfilter --question "Is this a programming language?" --keep-if no
```

Output:
```
English
Spanish
```