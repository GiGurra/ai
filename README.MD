# `ai` CLI Tool

A minimal cli client turning each terminal session into a persistent conversation with an LLM.

Conversation state is mapped to the terminal/shell session, and can be restored/loaded later and renamed to something more useful.

So far the following AI providers are supported:

* OpenAI
* Google Cloud Vertex AI
* Google AI Studio

## WARNING

- This tool is likely to significantly increase your AI provider bill :).
- I initially built this tool for fun, over the course of 3 evenings. It's a hack, treat it as such.

## Requirements

- An OpenAPI API key, or equivalent for other providers

## API

```
~> ai --help
ai/llm conversation tool, every terminal is a conversation

Usage:
  ai <question> [flags]
  ai [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Prints the current configuration
  copy        Copy a session
  delete      Delete a session, or the current session if no session id is provided
  help        Help about any command
  history     Prints the conversation history of the current session
  name-all    generate names to replace UUID session IDs
  new         Create a new session
  prep        Add a user message to the current session without sending a question
  rename      Rename a session
  reset       Create a new session
  session     Print id of current session
  sessions    List all stored sessions
  set         Set the ai session
  status      Prints info about current session

Flags:
      --verbose                   Verbose output (default false)
      --session string            Session id (deprecated) (env: CURRENT_AI_SESSION)
      --provider string           AI provider to use (env: AI_PROVIDER)
      --model string              Model to use
      --temperature float         Temperature to use
      --provider-api-key string   API key for provider (env: PROVIDER_API_KEY)
  -h, --help                      help for ai

Use "ai [command] --help" for more information about a command.
```

## Installation

```
go install github.com/gigurra/ai@latest
```

or:

1. Clone the repository:
    ```sh
    git clone https://github.com/gigurra/ai.git
    cd ai
    ```

2. Build the project:
    ```sh
    go build -o ai
    ```

3. Run the CLI tool:
    ```sh
    ./ai
    ```

Alternatively just install it with `go install .`, and run it with `ai` instead of `./ai`.

## Usage

### TLDR

```
 ~> ai please say hello
Hello! How can I assist you today?
 ~> ai session
4aca9b5b-deb2-4b67-bdc1-6ffb03515faf
 ~> ai rename my-session
 ~> ai session
my-session
 ~> ai sessions -v
my-session (i=10/10, o=9/9, created 2024-05-21 22:58:07)
dae13b8e-5602-448b-98d8-41e76b4b7cdd (i=32/42, o=9/18, created 2024-05-21 22:27:41)
c6381494-0b0f-4ae2-a5bc-d42cb52b37f3 (i=32/42, o=9/18, created 2024-05-21 22:23:22)
 ~> ai please translate that to danish
Hej! Hvordan kan jeg hjælpe dig i dag?
```

```
~> ll | ai remove all rows except those starting with g
Sure, here are the rows that start with the letter "g":
drwxr-xr-x@  6 johankjolhede  staff   192B 21 Maj 12:38 git
drwxr-xr-x@  4 johankjolhede  staff   128B 29 Nov 12:28 go
drwxr-xr-x@ 21 johankjolhede  staff   672B 18 Maj 13:34 google-cloud-sdk
drwxr-xr-x@ 21 johankjolhede  staff   672B 18 Okt  2023 google-cloud-sdk.bak
```

### Basic Commands

- **Ask a Question**:
    ```sh
    ai "What is the capital of France?"
    ```

- **List Sessions**:
    ```sh
    ai sessions
    ```

- **View Current Session**:
    ```sh
    ai session
    ```

- **View Session Status**:
    ```sh
    ai status
    ```

- **View Configuration**:
    ```sh
    ai config
    ```

- **View Conversation History**:
    ```sh
    ai history
    ```

### Session Management

- **Create a New Session**:
    ```sh
    ai new
    ```

- **Reset the Current Session**:
    ```sh
    ai reset
    ```

- **Set a Specific Session**:
    ```sh
    ai set <session_id>
    ```

- **Rename a Session**:
    ```sh
    ai rename <old_session_id> <new_session_id>
    ```

- **Copy a Session**:
    ```sh
    ai copy <source_session_id> <target_session_id>
    ```

- **Delete a Session**:
    ```sh
    ai delete <session_id>
    ```

### Using together with [aicat](https://github.com/gigurra/aicat)

You can use this tool together with cat or aicat to analyze a set of file.

- **Analyze a code base**:
    ```sh
    aicat . -p "*.go,*.js" | ai what do you think of this code?
    ```

- **Find where specific code is in a code base**:
    ```sh
    aicat . -p "*.go,*.js" | ai "where is the code that does X?"
    ```

- **Write a github readme**:
    ```sh
    aicat . -p "*.go,*.mod" | ai "please write a concise github readme explaining what this project does"
    ```

## Configuration

The configuration file is located at `~/.config/gigurra/ai/config.yaml`. It is created automatically if it does not
exist. You can configure the AI provider, model, and other settings in this file.

### Example Configuration

For OpenAI

```yaml
provider: openai
openai:
  api_key: "your_openai_api_key"
  model: "gpt-4o"
  temperature: 0.7
```

For Google Cloud Vertex AI (e.g. gemini-1.5-pro). This will authenticate by delegating
to `gcloud auth print-access-token` - a mechanism I will probably replace in the future.

```yaml
provider: google-cloud
google-cloud:
  project_id: "my-project-1"
  location_id: "europe-west4"
  model_id: "gemini-1.5-pro-001"
```

For Google AI Studio

```yaml
provider: google-ai-studio
google_ai_studio:
  api_key: "your-api-key"
  model_id: "gemini-1.5-pro-001"
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## Contact

For any questions or support, please open an issue on the GitHub repository.

---

Happy coding! 🚀
