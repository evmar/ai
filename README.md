A simple command-line wrapper for the OpenAI API and ollama.

There are probably better ones out there, but this one is mine!

## Setup

Requires a config file at `~/.config/ai.toml`.

Sample config file:

```
default_backend = "llama"

[backend.openai]
mode = "openai"

[backend.llama]
mode = "ollama"
model = "llama3.2:1b"
# if url unspecified, obeys $OLLAMA_HOST env, defaulting to localhost
url = "http://somehost:11434"
```
