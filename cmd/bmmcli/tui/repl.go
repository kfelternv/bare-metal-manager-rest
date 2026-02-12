// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"fmt"
	"os"
	"strings"
)

const maxSuggestions = 6

// argResourceMap maps command names to the resource type whose names should
// be offered as argument completions. Only commands that accept a resource
// name/ID as their first argument are listed here.
var argResourceMap = map[string]string{
	"site get":                     "site",
	"site delete":                  "site",
	"vpc get":                      "vpc",
	"vpc delete":                   "vpc",
	"subnet get":                   "subnet",
	"subnet delete":                "subnet",
	"instance get":                 "instance",
	"instance delete":              "instance",
	"allocation get":               "allocation",
	"allocation delete":            "allocation",
	"machine get":                  "machine",
	"ip-block get":                 "ip-block",
	"ip-block delete":              "ip-block",
	"operating-system get":         "operating-system",
	"ssh-key-group get":            "ssh-key-group",
	"network-security-group get":   "network-security-group",
}

// RunREPL starts the interactive REPL loop with inline autocomplete
func RunREPL(s *Session) error {
	commands := AllCommands()
	cmdNames := make([]string, len(commands))
	cmdMap := make(map[string]Command, len(commands))
	for i, cmd := range commands {
		cmdNames[i] = cmd.Name
		cmdMap[cmd.Name] = cmd
	}
	// Add exit/quit to suggestions but not cmdMap
	cmdNames = append(cmdNames, "exit", "quit")

	fmt.Printf("\n%s\n", Bold("BMM Interactive Mode"))
	fmt.Printf("Org: %s\n", Cyan(s.Org))
	fmt.Printf("Start typing a command. %s to quit.\n\n", Bold("Ctrl+D"))

	for {
		line, err := readLineWithSuggestions(s, cmdNames)
		if err != nil {
			// Ctrl+D or read error
			fmt.Println("\nGoodbye.")
			return nil
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "exit" || line == "quit" {
			fmt.Println("Goodbye.")
			return nil
		}

		// Try exact match first
		if cmd, ok := cmdMap[line]; ok {
			if err := cmd.Run(s, nil); err != nil {
				fmt.Fprintf(os.Stderr, "%s %v\n", Red("Error:"), err)
			}
			fmt.Println()
			continue
		}

		// Try matching with args
		matched := false
		for _, cmd := range commands {
			if strings.HasPrefix(line, cmd.Name) {
				rest := strings.TrimSpace(line[len(cmd.Name):])
				var args []string
				if rest != "" {
					args = strings.Fields(rest)
				}
				if err := cmd.Run(s, args); err != nil {
					fmt.Fprintf(os.Stderr, "%s %v\n", Red("Error:"), err)
				}
				fmt.Println()
				matched = true
				break
			}
		}

		if !matched {
			fmt.Fprintf(os.Stderr, "%s unknown command: %s\n", Red("Error:"), line)
			fmt.Println()
		}
	}
}

// readLineWithSuggestions reads a line in raw mode, showing matching suggestions
// below the input as the user types. It suggests command names first, and once
// the user has typed a complete command that accepts a resource argument (like
// "site get "), it switches to suggesting resource names from the cache.
func readLineWithSuggestions(s *Session, cmdNames []string) (string, error) {
	restore, err := RawMode()
	if err != nil {
		return "", err
	}
	defer func() {
		restore()
		ShowCursor()
	}()

	prompt := Cyan("bmm:"+s.Org) + "> "
	line := ""
	selectedSuggestion := -1 // -1 = no suggestion selected
	prevSuggestionCount := 0

	allSuggestions := func() []string {
		return getAllSuggestions(s, line, cmdNames)
	}

	renderInput := func() {
		suggestions := allSuggestions()
		if len(suggestions) > maxSuggestions {
			suggestions = suggestions[:maxSuggestions]
		}

		// Clear previous suggestions
		clearSuggestionLines(prevSuggestionCount)

		// Redraw the prompt line
		ClearLine()
		fmt.Print("\r" + prompt + line)

		// Clamp selected suggestion
		if selectedSuggestion >= len(suggestions) {
			selectedSuggestion = len(suggestions) - 1
		}

		// Draw suggestions below
		if len(line) > 0 && len(suggestions) > 0 {
			for i, sg := range suggestions {
				fmt.Print("\r\n")
				ClearLine()
				if i == selectedSuggestion {
					fmt.Print("  " + Reverse(" "+sg+" "))
				} else {
					fmt.Print("  " + Dim(sg))
				}
			}
			// Move cursor back up to the input line
			MoveUp(len(suggestions))
			// Position cursor at end of input
			MoveToColumn(len(stripAnsi(prompt)) + len(line) + 1)
		}

		prevSuggestionCount = len(suggestions)
		if len(line) == 0 {
			prevSuggestionCount = 0
		}
	}

	ShowCursor()
	renderInput()

	for {
		key, err := ReadKey()
		if err != nil {
			return "", err
		}

		switch {
		case key.Char == KeyCtrlC:
			// Clear and start over
			line = ""
			selectedSuggestion = -1
			clearSuggestionLines(prevSuggestionCount)
			prevSuggestionCount = 0
			renderInput()

		case key.Char == KeyCtrlD:
			clearSuggestionLines(prevSuggestionCount)
			return "", fmt.Errorf("EOF")

		case key.Char == KeyEnter || key.Char == KeyNewline:
			suggestions := allSuggestions()
			if len(suggestions) > maxSuggestions {
				suggestions = suggestions[:maxSuggestions]
			}

			// If a suggestion is selected, accept it
			if selectedSuggestion >= 0 && selectedSuggestion < len(suggestions) {
				line = suggestions[selectedSuggestion]
				selectedSuggestion = -1
				clearSuggestionLines(prevSuggestionCount)
				prevSuggestionCount = 0
				renderInput()
				continue
			}

			// Otherwise execute the line
			clearSuggestionLines(prevSuggestionCount)
			ClearLine()
			fmt.Print("\r" + prompt + line + "\r\n")
			return line, nil

		case key.Char == '\t':
			// Tab: accept top suggestion
			suggestions := allSuggestions()
			if len(suggestions) > 0 {
				idx := selectedSuggestion
				if idx < 0 {
					idx = 0
				}
				if idx < len(suggestions) {
					line = suggestions[idx]
					selectedSuggestion = -1
				}
			}
			renderInput()

		case key.Special == KeyDown:
			suggestions := allSuggestions()
			if len(suggestions) > maxSuggestions {
				suggestions = suggestions[:maxSuggestions]
			}
			if len(suggestions) > 0 {
				selectedSuggestion++
				if selectedSuggestion >= len(suggestions) {
					selectedSuggestion = 0
				}
			}
			renderInput()

		case key.Special == KeyUp:
			suggestions := allSuggestions()
			if len(suggestions) > maxSuggestions {
				suggestions = suggestions[:maxSuggestions]
			}
			if len(suggestions) > 0 {
				selectedSuggestion--
				if selectedSuggestion < 0 {
					selectedSuggestion = len(suggestions) - 1
				}
			}
			renderInput()

		case key.Char == KeyBackspace:
			if len(line) > 0 {
				line = line[:len(line)-1]
				selectedSuggestion = -1
			}
			renderInput()

		case key.Char >= 32 && key.Char < 127:
			line += string(key.Char)
			selectedSuggestion = -1
			renderInput()

		default:
			continue
		}
	}
}

// getAllSuggestions returns suggestions for the current input. If the input
// matches a command that accepts resource arguments (e.g. "site get "),
// resource names are suggested. Otherwise command names are suggested.
func getAllSuggestions(s *Session, input string, cmdNames []string) []string {
	if input == "" {
		return nil
	}

	// Check if input starts with a command that takes resource args + a space
	for cmdPrefix, resourceType := range argResourceMap {
		withSpace := cmdPrefix + " "
		if strings.HasPrefix(strings.ToLower(input), strings.ToLower(withSpace)) {
			argPart := input[len(withSpace):]
			return getResourceSuggestions(s, cmdPrefix, resourceType, argPart)
		}
	}

	// Fall back to command name suggestions
	return getCommandSuggestions(input, cmdNames)
}

// getResourceSuggestions returns resource names matching the typed argument,
// prefixed with the command so that accepting a suggestion fills the full line.
func getResourceSuggestions(s *Session, cmdPrefix, resourceType, argFilter string) []string {
	// Try to get items from cache (don't block on API calls during typing)
	items := s.Cache.Get(resourceType)
	if items == nil {
		// Cache is empty -- try a background fetch so next keystroke has data.
		// For now, fetch inline (it's fast for cached items).
		fetched, err := s.Resolver.Fetch(s.Ctx, resourceType)
		if err != nil {
			return nil
		}
		items = fetched
	}

	lowerFilter := strings.ToLower(argFilter)
	var matches []string
	for _, item := range items {
		name := item.Name
		if name == "" {
			name = item.ID
		}
		if lowerFilter == "" || strings.Contains(strings.ToLower(name), lowerFilter) {
			matches = append(matches, cmdPrefix+" "+name)
		}
	}
	return matches
}

// getCommandSuggestions returns command names matching the input prefix.
func getCommandSuggestions(input string, cmdNames []string) []string {
	lower := strings.ToLower(input)
	var matches []string
	for _, name := range cmdNames {
		if strings.HasPrefix(strings.ToLower(name), lower) {
			matches = append(matches, name)
		}
	}
	return matches
}

func clearSuggestionLines(count int) {
	if count == 0 {
		return
	}
	// Save current position, move down and clear each line, then return
	for i := 0; i < count; i++ {
		fmt.Print("\r\n")
		ClearLine()
	}
	MoveUp(count)
}

// stripAnsi removes ANSI escape codes from a string to get the visible length
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}
