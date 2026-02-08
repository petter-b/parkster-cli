package output

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Print outputs data in the specified format
func Print(data any, format string) error {
	switch format {
	case "json":
		return printJSON(data)
	case "tsv":
		return printTSV(data)
	default:
		return printPlain(data)
	}
}

func printJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func printTSV(data any) error {
	v := reflect.ValueOf(data)

	// Handle slices
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			if err := printTSVRow(v.Index(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	}

	return printTSVRow(data)
}

func printTSVRow(data any) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		fmt.Println(data)
		return nil
	}

	var fields []string
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanInterface() {
			continue
		}
		// Skip empty fields
		if isZero(field) {
			fields = append(fields, "")
			continue
		}
		fields = append(fields, fmt.Sprintf("%v", field.Interface()))
		_ = t.Field(i) // Could use for headers
	}

	fmt.Println(strings.Join(fields, "\t"))
	return nil
}

func printPlain(data any) error {
	v := reflect.ValueOf(data)

	// Handle slices
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			if err := printPlainItem(v.Index(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	}

	return printPlainItem(data)
}

func printPlainItem(data any) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Simple types
	if v.Kind() != reflect.Struct {
		fmt.Println(data)
		return nil
	}

	// Structs - print field: value
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanInterface() {
			continue
		}
		if isZero(field) {
			continue
		}

		name := t.Field(i).Name
		// Use json tag if available
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

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	}
	return false
}

// Table prints data as a formatted table
type Table struct {
	Headers []string
	Rows    [][]string
}

func (t *Table) Print() {
	if len(t.Rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = len(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range t.Headers {
		fmt.Printf("%-*s", widths[i]+2, h)
	}
	fmt.Println()

	// Print separator
	for _, w := range widths {
		fmt.Print(strings.Repeat("-", w+2))
	}
	fmt.Println()

	// Print rows
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("%-*s", widths[i]+2, cell)
			}
		}
		fmt.Println()
	}
}
