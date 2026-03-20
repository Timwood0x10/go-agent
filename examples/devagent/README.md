# DevAgent - Developer Assistant with Multi-Agent Orchestration

A command-line developer assistant that uses multi-agent orchestration to help with code generation, review, testing, and documentation. **Creates actual files** - not just text output!

## Features

- **Multi-Agent Orchestration**: Leader agent coordinates specialized sub-agents
- **DAG-based Workflow**: Parallel execution of independent tasks
- **Agent Communication**: Sub-agents can communicate via message queue
- **Memory Distillation**: Task history and user profile tracking
- **Interactive CLI**: User-friendly command-line interface
- **Automatic File Generation**: Creates actual code files, tests, and documentation
- **Real Development Tool**: Produces production-ready artifacts, not just text

## Agent Roles

### Leader Agent
- **Model**: `meta-llama/llama-3.1-8b-instruct` (free, lightweight, fast)
- **Responsibilities**: Task analysis, decomposition, and coordination
- **Actions**: Analyzes user input, breaks down tasks, dispatches to sub-agents

### Code Agent
- **Model**: `allenai/olmo-3.1-32b-think` (free, good for code generation)
- **Responsibilities**: Generate high-quality code implementations
- **Specializations**: API development, algorithms, data structures, utilities

### Review Agent
- **Model**: `google/gemini-3.1-flash-lite-preview` (free, efficient review)
- **Responsibilities**: Review generated code for quality, security, and best practices
- **Dependencies**: Requires code and documentation outputs

### Test Agent
- **Model**: `google/gemini-3.1-flash-lite-preview` (free, efficient test generation)
- **Responsibilities**: Generate unit tests and integration tests
- **Dependencies**: Requires code output

### Docs Agent
- **Model**: `google/gemini-3.1-flash-lite-preview` (free, clear documentation)
- **Responsibilities**: Generate API documentation, README, and usage examples
- **Parallel Execution**: Can run concurrently with code generation

## Workflow Orchestration

The workflow uses DAG (Directed Acyclic Graph) orchestration:

```
analyze (Leader)
  ├── code (Code Agent) ─┬── test (Test Agent)
  └── docs (Docs Agent) ─┴── review (Review Agent)
```

**Key Features**:
- Code and Docs can execute in parallel
- Test depends on Code
- Review depends on both Code and Docs
- Efficient task execution and coordination

## Prerequisites

1. **OpenRouter API Key**: Get your API key from https://openrouter.ai/
2. **Go 1.26+**: Go runtime environment
3. **Configuration**: Set up your `config/server.yaml`

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd styleagent
```

2. Install dependencies:
```bash
go mod download
```

3. Configure API key:
```bash
export OPENROUTER_API_KEY="sk-or-v1-your-api-key"
```

Or set it in `config/server.yaml`:
```yaml
llm:
  api_key: "sk-or-v1-your-api-key"
```

## Usage

### Start the Assistant

```bash
go run examples/devagent/main.go
```

### Interactive Commands

Once started, you can interact with the assistant:

```
DevAgent> Create a REST API for user management in Python
DevAgent> Implement a binary search algorithm in Go
DevAgent> Write unit tests for a sorting function
DevAgent> Generate documentation for a data processing pipeline
DevAgent> help        # Show help information
DevAgent> exit        # Exit the assistant
```

### Example Session

```
DevAgent> Create a REST API for user management in Python

Processing: Create a REST API for user management in Python
--------------------------------------------------
✅ Successfully created 4 file(s):
   📄 user_api.py
   📄 README.md
   📄 test_user_api.py
   📄 REVIEW_user_api.md

Generated 4 item(s):

[1] User Management API
    Type: code
    Description: RESTful API with CRUD operations for user management
    Price: 0.00

[2] API Documentation
    Type: docs
    Description: Complete API documentation with examples
    Price: 0.00

[3] Unit Tests
    Type: test
    Description: Comprehensive test suite for user API
    Price: 0.00

[4] Code Review
    Type: review
    Description: Code quality and security review
    Price: 0.00

Summary: 4 items generated
Completed in 12.5s

--------------------------------------------------
DevAgent> exit
Goodbye!
```

### File Creation

The agent automatically creates files in the current directory:

- **Code files**: `.py` files with actual implementation
- **Test files**: `test_*.py` files with unit tests
- **Documentation**: `README.md` or `*.md` files with documentation
- **Reviews**: `REVIEW_*.md` files with code review results

All files are automatically saved with appropriate permissions.

## Configuration

### Main Configuration (`config/server.yaml`)

```yaml
llm:
  provider: "openrouter"
  api_key: "your-api-key"
  model: "meta-llama/llama-3.1-8b-instruct"

agents:
  leader:
    id: "leader-dev"
    max_parallel_tasks: 4

  sub:
    - id: "agent-code"
      type: "code"
      model: "allenai/olmo-3.1-32b-think"
    
    - id: "agent-review"
      type: "review"
      model: "google/gemini-3.1-flash-lite-preview"
    
    - id: "agent-test"
      type: "test"
      model: "google/gemini-3.1-flash-lite-preview"
    
    - id: "agent-docs"
      type: "docs"
      model: "google/gemini-3.1-flash-lite-preview"

memory:
  enabled: true
  task_distillation:
    enabled: true
```

### Workflow Configuration (`config/workflow.yaml`)

Defines the DAG orchestration:

```yaml
steps:
  - id: "analyze"
    agent_type: "leader"
  
  - id: "code"
    agent_type: "code"
    depends_on: ["analyze"]
  
  - id: "docs"
    agent_type: "docs"
    depends_on: ["analyze"]
  
  - id: "test"
    agent_type: "test"
    depends_on: ["code"]
  
  - id: "review"
    agent_type: "review"
    depends_on: ["code", "docs"]
```

## Memory and Distillation

The assistant includes memory features:

- **Session History**: Tracks conversation context (100 turns)
- **User Profile**: Learns user preferences and patterns
- **Task Distillation**: Summarizes key information from tasks

Configuration:
```yaml
memory:
  session:
    max_history: 100
  
  task_distillation:
    enabled: true
    prompt: "Summarize key information: task type, language, requirements, features"
```

## Performance

- **Task Analysis**: ~2-3 seconds (Llama 3.1 8B)
- **Code Generation**: ~5-8 seconds (OLMo 3.1 32B)
- **Documentation**: ~3-5 seconds (parallel with code, Gemini Flash Lite)
- **Test Generation**: ~3-5 seconds (after code, Gemini Flash Lite)
- **Code Review**: ~3-5 seconds (after code and docs, Gemini Flash Lite)

**Total Time**: ~8-12 seconds for complete workflow (parallel execution with free models)

## Code Quality

This example follows the project's coding standards:

- ✅ Proper error handling
- ✅ Context propagation
- ✅ Concurrent safety
- ✅ Clean architecture
- ✅ Comprehensive documentation
- ✅ Type safety

## Limitations

1. **Single Session**: No persistent storage between sessions
2. **No Build Tools**: Cannot compile or execute generated code
3. **Simple Dependencies**: Basic DAG without complex branching
4. **Local Only**: No remote repository integration

## Future Enhancements

- [ ] Git integration for version control
- [ ] Automatic code execution and testing
- [ ] Multi-file project generation
- [ ] Dependency management (requirements.txt, go.mod)
- [ ] Integration with IDE plugins
- [ ] Advanced dependency resolution
- [ ] Performance metrics and monitoring
- [ ] Project scaffolding from templates

## Troubleshooting

### API Key Issues

```
Error: Failed to load config: invalid API key
```

**Solution**: Ensure your OpenRouter API key is set correctly in environment variable or config file.

### Model Access Issues

```
Error: Failed to create adapter
```

**Solution**: Check that you have access to the specified models in your OpenRouter account.

### Timeout Issues

```
Error: context deadline exceeded
```

**Solution**: Increase timeout values in config/server.yaml for complex tasks.

## Support

For issues or questions:
1. Check the main project README
2. Review configuration files
3. Check OpenRouter API status
4. Review logs for detailed error messages

## License

Same as the main StyleAgent project.