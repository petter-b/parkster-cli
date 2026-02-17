package output

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Mode determines how output is formatted
type Mode int

const (
	ModeHuman Mode = iota // default: human-readable
	ModeJSON              // --json: JSON with envelope
)

// Envelope is the JSON output wrapper
type Envelope struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
	Error   any  `json:"error"`
}

// ModeFromFlags returns the output mode based on CLI flags
func ModeFromFlags(jsonFlag bool) Mode {
	if jsonFlag {
		return ModeJSON
	}
	return ModeHuman
}

// PrintSuccess outputs data in the specified mode
func PrintSuccess(data any, mode Mode) error {
	switch mode {
	case ModeJSON:
		return printJSONEnvelope(true, data, nil)
	default:
		return printHuman(data)
	}
}

// PrintError outputs an error in the specified mode.
// In JSON mode, prints envelope to stdout. In human mode, prints to stderr.
func PrintError(msg string, mode Mode) {
	switch mode {
	case ModeJSON:
		_ = printJSONEnvelope(false, nil, msg)
	default:
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
}

func printJSONEnvelope(success bool, data any, errMsg any) error {
	env := Envelope{
		Success: success,
		Data:    data,
		Error:   errMsg,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

// --- Human-readable output (field: value) ---

func printHuman(data any) error {
	v := reflect.ValueOf(data)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Handle slices
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				fmt.Println()
			}
			if err := printHumanItem(v.Index(i).Interface()); err != nil {
				return fmt.Errorf("printing item %d: %w", i, err)
			}
		}
		return nil
	}

	return printHumanItem(data)
}

func printHumanItem(data any) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		fmt.Println(data)
		return nil
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanInterface() {
			continue
		}

		name := t.Field(i).Name
		if tag := t.Field(i).Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		}

		fmt.Printf("%s: %v\n", name, field.Interface())
	}
	return nil
}
