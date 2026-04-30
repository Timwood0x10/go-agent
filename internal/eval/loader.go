package eval

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Loader loads test suites from files.
type Loader struct{}

// NewLoader creates a new test suite loader.
func NewLoader() *Loader {
	return &Loader{}
}

// Load loads a test suite from a YAML file.
func (l *Loader) Load(path string) (*TestSuite, error) {
	// Validate path to prevent directory traversal
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid path: directory traversal not allowed")
	}

	data, err := os.ReadFile(path) // #nosec G304 -- Path validated above to prevent traversal
	if err != nil {
		return nil, err
	}

	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, err
	}

	// Set default timeout for test cases without timeout
	for i := range suite.TestCases {
		if suite.TestCases[i].Timeout == 0 {
			suite.TestCases[i].Timeout = 30 * time.Second
		}
	}

	return &suite, nil
}

// LoadDir loads all test suites from a directory.
func (l *Loader) LoadDir(dir string) ([]TestSuite, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var suites []TestSuite
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only load YAML files
		name := entry.Name()
		if len(name) < 5 || (name[len(name)-5:] != ".yaml" && name[len(name)-4:] != ".yml") {
			continue
		}

		path := dir + "/" + name
		suite, err := l.Load(path)
		if err != nil {
			return nil, err
		}
		suites = append(suites, *suite)
	}

	return suites, nil
}
