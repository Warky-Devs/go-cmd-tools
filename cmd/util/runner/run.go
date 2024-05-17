package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type Command struct {
	Name        string            `json:"name"`
	Arguments   []string          `json:"arguments"`
	Environment map[string]string `json:"environment,omitempty"`
	Description string            `json:"description,omitempty"`
	WorkingDir  string            `json:"cwd,omitempty"`
}

const (
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return
	}
	fmt.Print("\n------------------------------------------------------------------------\n")
	fmt.Printf("🐒 Runner Started in %s", cwd)
	fmt.Print("\n------------------------------------------------------------------------\n")
	// Read commands from JSON file
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go commands.json")
		return
	}

	// Read commands from JSON file
	commands, err := readCommandsFromFile(os.Args[1])
	if err != nil {
		fmt.Println("Error reading commands from file:", err)
		return
	}
	// Create a wait group to wait for all commands to finish
	var wg sync.WaitGroup
	wg.Add(len(commands))

	// Channel to receive errors from goroutines
	errorCh := make(chan error, len(commands))

	// Channel to indicate completion of each command
	doneCh := make(chan string, len(commands))

	// Loop through each command and execute them concurrently
	for _, cmd := range commands {
		go func(cmd Command) {
			defer wg.Done()
			fmt.Print("\n🐒------------------------------------------------------------------------🐒\n")
			fmt.Printf("Running:  %s\n", cmd.Description)
			fmt.Printf("Directory: %s\n", cmd.WorkingDir)
			fmt.Printf("Cmd: %s %v\n", cmd.Name, cmd.Arguments)
			fmt.Print("\n🐒------------------------------------------------------------------------🐒\n")

			// Create the command
			c := exec.Command(cmd.Name, cmd.Arguments...)

			// Set environment variables if needed
			for key, value := range cmd.Environment {
				c.Env = append(c.Env, fmt.Sprintf("%s=%s", key, value))
			}

			// Set the working directory if specified
			if cmd.WorkingDir != "" {
				// Get the full path for the working directory if it's a partial path
				if cmd.WorkingDir != "" && !filepath.IsAbs(cmd.WorkingDir) {
					fullPath, err := filepath.Abs(cmd.WorkingDir)
					if err != nil {
						errorCh <- fmt.Errorf("🥺 error getting full path for working directory '%s': %w", cmd.WorkingDir, err)
						return
					}
					cmd.WorkingDir = fullPath
				}
				c.Dir = cmd.WorkingDir

			}

			// Start the command
			output, err := c.CombinedOutput()
			if err != nil {
				errorCh <- fmt.Errorf("exec error: '%s %v': \n%w \n%s", cmd.Name, cmd.Arguments, err, output)
				return
			}

			// Print command output
			fmt.Printf("Output of '%s %v':\n%s\n", cmd.Name, cmd.Arguments, output)
			doneCh <- fmt.Sprintf("%s %v", cmd.Name, cmd.Arguments)
		}(cmd)
	}

	// Close the error channel after all commands are finished
	go func() {
		wg.Wait()
		close(errorCh)
		close(doneCh)
	}()

	// Monitor error channel for any errors
	for err := range errorCh {
		fmt.Println("Error:", err)
		fmt.Printf("%s%v %s\n", colorRed, err, colorReset)
	}

	// Monitor done channel for command completions
	completed := make(map[string]bool)
	for doneCmd := range doneCh {
		completed[doneCmd] = true
	}

	// Check status of each command
	fmt.Print("\n-------------------------Commands Summary-----------------------------------\n")

	for _, cmd := range commands {
		if cmd.Description == "" {
			cmd.Description = fmt.Sprintf("%s %v", cmd.Name, cmd.Arguments)
		}
		cmdStr := fmt.Sprintf("%s %v", cmd.Name, cmd.Arguments)
		if completed[cmdStr] {
			fmt.Printf("%s ✔️  %s %s \n", colorGreen, cmd.Description, colorReset)
		} else {
			fmt.Printf("%s ❌  Error in %s:%s %s\n", colorRed, cmd.Description, colorReset, cmdStr)
		}
	}
	fmt.Print("\n-----------------------------------Done----------------------------------------\n")
}

// Function to read commands from JSON file
func readCommandsFromFile(filename string) ([]Command, error) {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read file contents
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON into a slice of commands
	var commands []Command
	if err := json.Unmarshal(bytes, &commands); err != nil {
		return nil, err
	}

	return commands, nil
}
