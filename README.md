# Go Claude Skills Parser

A Go package to parse Claude Skill packages from a directory structure. This parser is designed according to the specifications found in the [official Claude documentation](https://docs.claude.com/en/docs/agents-and-tools/agent-skills/).

[![License](https://img.shields.io/:license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![GoDoc](https://godoc.org/github.com/smallnest/goskills?status.png)](http://godoc.org/github.com/smallnest/goskills)  [![github actions](https://github.com/smallnest/goskills/actions)](https://github.com/smallnest/goskills/actions/workflows/go.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/smallnest/goskills)](https://goreportcard.com/report/github.com/smallnest/goskills) 

## Features

- Parses `SKILL.md` for skill metadata and instructions.
- Extracts YAML frontmatter into a Go struct (`SkillMeta`).
- Captures the Markdown body of the skill.
- Discovers resource files in `scripts/`, `references/`, and `assets/` directories.
- Packaged as a reusable Go module.
- Includes command-line interfaces for managing and inspecting skills.

## Installation

To use this package in your project, you can use `go get`:

```shell
go get github.com/smallnest/goskills
```

## Library Usage

Here is an example of how to use the `ParseSkillPackage` function from the `goskills` library to parse a skill directory.

```go
package main

import (
	"fmt"
	"log"

	"github.com/smallnest/goskills"
)

func main() {
	// Path to the skill directory you want to parse
	skillDirectory := "./examples/skills/artifacts-builder"

	skillPackage, err := goskills.ParseSkillPackage(skillDirectory)
	if err != nil {
		log.Fatalf("Failed to parse skill package: %v", err)
	}

	// Print the parsed information
	fmt.Printf("Successfully Parsed Skill: %s\n", skillPackage.Meta.Name)
	// ... and so on
}
```

### ParseSkillPackages

To find and parse all valid skill packages within a directory and its subdirectories, you can use the `ParseSkillPackages` function. It recursively scans the given path, identifies all directories containing a `SKILL.md` file, and returns a slice of successfully parsed `*SkillPackage` objects.

```go
package main

import (
	"fmt"
	"log"

	"github.com/smallnest/goskills"
)

func main() {
	// Directory containing all your skills
	skillsRootDirectory := "./examples/skills"

	packages, err := goskills.ParseSkillPackages(skillsRootDirectory)
	if err != nil {
		log.Fatalf("Failed to parse skill packages: %v", err)
	}

	fmt.Printf("Found %d skill(s):\n", len(packages))
	for _, pkg := range packages {
		fmt.Printf("- Path: %s, Name: %s\n", pkg.Path, pkg.Meta.Name)
	}
}
```

## Command-Line Interfaces

This project provides two separate command-line tools:

### 1. Skill Management CLI (`goskills-cli`)

Located in `cmd/skill-cli`, this tool helps you inspect and manage your local Claude skills.

#### Building `goskills-cli`
You can build the executable from the project root:
```shell
go build -o goskills-cli ./cmd/skill-cli
```

#### Commands
Here are the available commands for `goskills-cli`:

#### list
Lists all valid skills in a given directory.
```shell
./goskills-cli list ./examples/skills
```

#### parse
Parses a single skill and displays a summary of its structure.
```shell
./goskills-cli parse ./examples/skills/artifacts-builder
```

#### detail
Displays the full, detailed information for a single skill, including the complete body content.
```shell
./goskills-cli detail ./examples/skills/artifacts-builder
```

#### files
Lists all the files that make up a skill package.
```shell
./goskills-cli files ./examples/skills/artifacts-builder
```

#### search
Searches for skills by name or description within a directory. The search is case-insensitive.
```shell
./goskills-cli search ./examples/skills "web app"
```

### 2. Skill Runner CLI (`goskills-runner`)

Located in `cmd/skill-runner`, this tool simulates the Claude skill-use workflow by integrating with Large Language Models (LLMs) like OpenAI's models.

#### Building `goskills-runner`
You can build the executable from the project root:
```shell
go build -o goskills-runner ./cmd/skill-runner
```

#### Commands
Here are the available commands for `goskills-runner`:

#### run
Processes a user request by first discovering available skills, then asking an LLM to select the most appropriate one, and finally executing the selected skill by feeding its content to the LLM as a system prompt.

**Requires the `OPENAI_API_KEY` environment variable to be set.**

```shell
# Example with default OpenAI model (gpt-4o)
export OPENAI_API_KEY="YOUR_OPENAI_API_KEY"
./goskills-runner run "create an algorithm that generates abstract art"

# Example with a custom OpenAI-compatible model and API base URL using environment variables
export OPENAI_API_KEY="YOUR_OPENAI_API_KEY"
export OPENAI_API_BASE="https://qianfan.baidubce.com/v2"
export OPENAI_MODEL="deepseek-v3"
./goskills-runner run "create an algorithm that generates abstract art"

# Example with a custom OpenAI-compatible model and API base URL using command-line flags
export OPENAI_API_KEY="YOUR_OPENAI_API_KEY"
./goskills-runner run --model deepseek-v3 --api-base https://qianfan.baidubce.com/v2 "create an algorithm that generates abstract art"
```

## Running Tests

To run the tests for this package, navigate to the project root directory and run:

```shell
go test
```
