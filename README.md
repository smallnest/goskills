# Go Claude Skills Parser

A Go package to parse Claude Skill packages from a directory structure. This parser is designed according to the specifications found in the [official Claude documentation](https://docs.claude.com/en/docs/agents-and-tools/agent-skills/).

## Features

- Parses `SKILL.md` for skill metadata and instructions.
- Extracts YAML frontmatter into a Go struct (`SkillMeta`).
- Captures the Markdown body of the skill.
- Discovers resource files in `scripts/`, `references/`, and `assets/` directories.
- Packaged as a reusable Go module.
- Includes a command-line interface (`skill-cli`) for managing and inspecting skills.

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

## Command-Line Interface (skill-cli)

This project includes a standalone CLI tool for inspecting skills, located in the `cmd/skill-cli` directory.

### Building the CLI

You can build the executable from the project root:
```shell
go build -o goskills-cli ./cmd/skill-cli
```

### Commands

Here are the available commands:

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

## Running Tests

To run the tests for this package, navigate to the project root directory and run:

```shell
go test
```