package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Input struct {
		File1 string `yaml:"file1"`
		File2 string `yaml:"file2"`
	} `yaml:"input"`
}

type CompareResult struct {
	FalseCount int
}

func main() {
	config, err := loadConfig("config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	diff, result, err := compareJSON(config.Input.File1, config.Input.File2)
	if err != nil {
		log.Fatalf("Error comparing JSON files: %v", err)
	}

	printComparisonResult(diff, result)
}

func loadConfig(filePath string) (Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Config{}, fmt.Errorf("error opening config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("error decoding YAML: %w", err)
	}
	return config, nil
}

func readFileAndUnmarshal(filePath string) (interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON from file %s: %w", filePath, err)
	}

	return obj, nil
}

func compareJSON(file1Path, file2Path string) (string, CompareResult, error) {
	obj1, err := readFileAndUnmarshal(file1Path)
	if err != nil {
		return "", CompareResult{}, err
	}

	obj2, err := readFileAndUnmarshal(file2Path)
	if err != nil {
		return "", CompareResult{}, err
	}

	result := CompareResult{}
	diff := compareMaps(obj1, obj2, "", &result)
	if diff != "" {
		return fmt.Sprintf("Differences found:\n%s", diff), result, nil
	}

	return "JSON files are identical", result, nil
}

func compareMaps(m1, m2 interface{}, path string, result *CompareResult) string {
	switch v1 := m1.(type) {
	case map[string]interface{}:
		return compareMapObjects(v1, m2, path, result)
	case []interface{}:
		return compareArrayObjects(v1, m2, path, result)
	default:
		return comparePrimitiveObjects(v1, m2, path, result)
	}
}

func compareMapObjects(m1 map[string]interface{}, m2 interface{}, path string, result *CompareResult) string {
	v2, ok := m2.(map[string]interface{})
	if !ok {
		result.FalseCount++
		return fmt.Sprintf("Type mismatch at %s: expected map[string]interface{} got %T\n", path, m2)
	}

	var diff string
	for key, val1 := range m1 {
		newPath := joinPath(path, key)
		val2, ok := v2[key]
		if !ok {
			diff += fmt.Sprintf("Key '%s' missing in second map at %s\n", key, newPath)
			result.FalseCount++
			continue
		}
		subDiff := compareMaps(val1, val2, newPath, result)
		if subDiff != "" {
			diff += subDiff
		}
	}
	for key := range v2 {
		if _, ok := m1[key]; !ok {
			newPath := joinPath(path, key)
			diff += fmt.Sprintf("Key '%s' missing in first map at %s\n", key, newPath)
			result.FalseCount++
		}
	}
	return diff
}

func compareArrayObjects(a1 []interface{}, m2 interface{}, path string, result *CompareResult) string {
	v2, ok := m2.([]interface{})
	if !ok {
		result.FalseCount++
		return fmt.Sprintf("Type mismatch at %s: expected []interface{} got %T\n", path, m2)
	}

	if len(a1) != len(v2) {
		result.FalseCount++
		return fmt.Sprintf("Length mismatch at %s: %d != %d\n", path, len(a1), len(v2))
	}

	var diff string
	for i := range a1 {
		newPath := fmt.Sprintf("%s[%d]", path, i)
		subDiff := compareMaps(a1[i], v2[i], newPath, result)
		if subDiff != "" {
			diff += subDiff
		}
	}
	return diff
}

func comparePrimitiveObjects(v1, v2 interface{}, path string, result *CompareResult) string {
	if v1 != v2 {
		result.FalseCount++
		return fmt.Sprintf("Value mismatch at %s: %v != %v\n", path, v1, v2)
	}
	return ""
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

func printComparisonResult(diff string, result CompareResult) {
	fmt.Println(diff)
	fmt.Printf("Compare False Count: %d\n", result.FalseCount)
}
