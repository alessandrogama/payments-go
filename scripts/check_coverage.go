package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	coverageFile := "coverage.out"
	targetPercent := 80.0

	if len(os.Args) >= 2 {
		coverageFile = os.Args[1]
	}
	if len(os.Args) >= 3 {
		val, err := strconv.ParseFloat(os.Args[2], 64)
		if err == nil {
			targetPercent = val
		}
	}

	fmt.Printf("Analyzing coverage file: %s (Target: %.2f%%)\n", coverageFile, targetPercent)

	// Filter out boilerplate setup files from coverage
	file, err := os.Open(coverageFile)
	if err != nil {
		fmt.Printf("Error opening coverage file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	filteredPath := "coverage_filtered.out"
	filteredFile, err := os.Create(filteredPath)
	if err != nil {
		fmt.Printf("Error creating filtered coverage file: %v\n", err)
		os.Exit(1)
	}
	defer filteredFile.Close()

	ignoredPatterns := []string{
		"postgres.go",
		"redis.go",
		"router.go",
		"mocks.go",
	}

	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(filteredFile)
	
	// Keep the mode line (first line of go coverage profile, e.g., "mode: set")
	if scanner.Scan() {
		_, _ = writer.WriteString(scanner.Text() + "\n")
	}

	for scanner.Scan() {
		line := scanner.Text()
		shouldIgnore := false
		for _, pat := range ignoredPatterns {
			if strings.Contains(line, pat) {
				shouldIgnore = true
				break
			}
		}
		if !shouldIgnore {
			_, _ = writer.WriteString(line + "\n")
		}
	}
	_ = writer.Flush()

	// Run "go tool cover -func=coverage_filtered.out"
	cmd := exec.Command("go", "tool", "cover", "-func="+filteredPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error running go tool cover: %v\n", err)
		fmt.Println(stderr.String())
		os.Exit(1)
	}

	// Read lines to find the "total:" line
	funcScanner := bufio.NewScanner(&stdout)
	totalRe := regexp.MustCompile(`total:\s+\(statements\)\s+([\d.]+)%`)
	var foundPercent float64
	var foundTotalLine bool

	for funcScanner.Scan() {
		line := funcScanner.Text()
		matches := totalRe.FindStringSubmatch(line)
		if len(matches) == 2 {
			val, err := strconv.ParseFloat(matches[1], 64)
			if err == nil {
				foundPercent = val
				foundTotalLine = true
				fmt.Println(line)
			}
		}
	}

	if !foundTotalLine {
		fmt.Println("Error: Could not parse overall coverage from 'go tool cover' output.")
		os.Exit(1)
	}

	if foundPercent < targetPercent {
		fmt.Printf("FAIL: Code coverage of %.2f%% is below the required minimum of %.2f%%\n", foundPercent, targetPercent)
		os.Exit(1)
	}

	fmt.Printf("SUCCESS: Code coverage is %.2f%%, which meets or exceeds the required minimum of %.2f%%\n", foundPercent, targetPercent)
}
