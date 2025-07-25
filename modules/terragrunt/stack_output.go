package terragrunt

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/gruntwork-io/terratest/modules/testing"
)

// TgOutput calls terragrunt stack output for the given variable and returns its value as a string
func TgOutput(t testing.TestingT, options *Options, key string) string {
	out, err := TgOutputE(t, options, key)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TgOutputE calls terragrunt stack output for the given variable and returns its value as a string
func TgOutputE(t testing.TestingT, options *Options, key string) (string, error) {
	rawOutput, err := runTerragruntStackCommandE(t, options, "output", outputArgs(options, key)...)
	if err != nil {
		return "", err
	}
	cleaned, err := cleanTerragruntOutput(rawOutput)
	if err != nil {
		return "", err
	}
	return cleaned, nil
}

// TgOutputJson calls terragrunt stack output for the given variable and returns the
// result as the json string.
// If key is an empty string, it will return all the output variables.
func TgOutputJson(t testing.TestingT, options *Options, key string) string {
	str, err := TgOutputJsonE(t, options, key)
	if err != nil {
		t.Fatal(err)
	}
	return str
}

// TgOutputJsonE calls terragrunt stack output for the given variable and returns the
// result as the json string.
// If key is an empty string, it will return all the output variables.
func TgOutputJsonE(t testing.TestingT, options *Options, key string) (string, error) {
	args := outputArgs(options, key)
	// Add -json flag for JSON output
	jsonArgs := append([]string{"-json"}, args...)

	rawOutput, err := runTerragruntStackCommandE(t, options, "output", jsonArgs...)
	if err != nil {
		return "", err
	}
	return cleanTerragruntJson(rawOutput)
}

// outputArgs builds the argument list for terragrunt stack output command
func outputArgs(options *Options, key string) []string {
	args := []string{"-no-color"}

	// Add all user-specified terragrunt command-line arguments first
	args = append(args, options.ExtraArgs...)

	// Add the key last, if provided
	if key != "" {
		args = append(args, key)
	}

	return args
}

const skipJsonLogLine = " msg="

var (
	// tgLogLevel matches log lines containing fields for time, level, prefix, binary, and message
	tgLogLevel = regexp.MustCompile(`.*time=\S+ level=\S+ prefix=\S+ binary=\S+ msg=.*`)
)

// cleanTerragruntOutput extracts the actual output value from terragrunt stack's verbose output
//
// Example input (raw terragrunt output):
//
//	time=2023-07-11T10:30:45Z level=info prefix=terragrunt binary=terragrunt msg="Initializing..."
//	time=2023-07-11T10:30:46Z level=info prefix=terragrunt binary=terragrunt msg="Running command..."
//	"my-bucket-name"
//
// Example output (cleaned):
//
//	my-bucket-name
//
// For JSON values, it preserves the structure:
// Input:
//
//	time=2023-07-11T10:30:45Z level=info prefix=terragrunt binary=terragrunt msg="Running..."
//	{"vpc_id": "vpc-12345", "subnet_ids": ["subnet-1", "subnet-2"]}
//
// Output:
//
//	{"vpc_id": "vpc-12345", "subnet_ids": ["subnet-1", "subnet-2"]}
func cleanTerragruntOutput(rawOutput string) (string, error) {
	// Remove terragrunt log lines
	cleaned := tgLogLevel.ReplaceAllString(rawOutput, "")

	lines := strings.Split(cleaned, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.Contains(trimmed, skipJsonLogLine) {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return "", nil
	}

	// Join all result lines
	finalOutput := strings.Join(result, "\n")

	// Check if it's JSON (starts with { or [)
	finalOutput = strings.TrimSpace(finalOutput)
	if strings.HasPrefix(finalOutput, "{") || strings.HasPrefix(finalOutput, "[") {
		// For JSON output, return as-is
		return finalOutput, nil
	}

	// For simple values, remove surrounding quotes if present
	if strings.HasPrefix(finalOutput, "\"") && strings.HasSuffix(finalOutput, "\"") {
		finalOutput = strings.Trim(finalOutput, "\"")
	}

	return finalOutput, nil
}

// cleanTerragruntJson cleans the JSON output from terragrunt stack command
//
// Example input (raw terragrunt JSON output):
//
//	time=2023-07-11T10:30:45Z level=info prefix=terragrunt binary=terragrunt msg="Initializing..."
//	time=2023-07-11T10:30:46Z level=info prefix=terragrunt binary=terragrunt msg="Running command..."
//	{"mother.output":{"sensitive":false,"type":"string","value":"mother/test.txt"},"father.output":{"sensitive":false,"type":"string","value":"father/test.txt"}}
//
// Example output (cleaned and formatted):
//
//	{
//	  "mother.output": {
//	    "sensitive": false,
//	    "type": "string",
//	    "value": "mother/test.txt"
//	  },
//	  "father.output": {
//	    "sensitive": false,
//	    "type": "string",
//	    "value": "father/test.txt"
//	  }
//	}
func cleanTerragruntJson(input string) (string, error) {
	// Remove terragrunt log lines
	cleaned := tgLogLevel.ReplaceAllString(input, "")

	lines := strings.Split(cleaned, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.Contains(trimmed, skipJsonLogLine) {
			result = append(result, trimmed)
		}
	}
	ansiClean := strings.Join(result, "\n")

	var jsonObj interface{}
	if err := json.Unmarshal([]byte(ansiClean), &jsonObj); err != nil {
		return "", err
	}

	// Format JSON output with indentation
	normalized, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		return "", err
	}

	return string(normalized), nil
}
