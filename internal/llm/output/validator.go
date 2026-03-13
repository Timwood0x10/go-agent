package output

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"goagent/internal/core/models"
)

// Validator validates data against schemas.
type Validator struct {
	customValidators map[string]ValidatorFunc
}

// ValidatorFunc is a custom validation function.
type ValidatorFunc func(interface{}) error

// NewValidator creates a new Validator.
func NewValidator() *Validator {
	v := &Validator{
		customValidators: make(map[string]ValidatorFunc),
	}

	v.registerDefaults()

	return v
}

// registerDefaults registers built-in validators.
func (v *Validator) registerDefaults() {
	v.RegisterValidator("string", v.validateString)
	v.RegisterValidator("number", v.validateNumber)
	v.RegisterValidator("integer", v.validateInteger)
	v.RegisterValidator("boolean", v.validateBoolean)
	v.RegisterValidator("array", v.validateArray)
	v.RegisterValidator("object", v.validateObject)
}

// RegisterValidator registers a custom validator.
func (v *Validator) RegisterValidator(name string, fn ValidatorFunc) {
	v.customValidators[name] = fn
}

// Validate validates data against a schema.
func (v *Validator) Validate(data interface{}, schema *Schema) error {
	if schema == nil {
		return nil
	}

	return v.validateValue(data, schema, "root")
}

func (v *Validator) validateValue(data interface{}, schema *Schema, path string) error {
	// Handle null
	if data == nil {
		if schema.Nullable {
			return nil
		}
		return fmt.Errorf("%s: value is null", path)
	}

	// Type validation
	if schema.Type != "" {
		if err := v.validateType(data, schema.Type, path); err != nil {
			return err
		}
	}

	// Enum validation
	if len(schema.Enum) > 0 {
		if err := v.validateEnum(data, schema.Enum, path); err != nil {
			return err
		}
	}

	// String-specific validations
	if str, ok := data.(string); ok {
		if schema.MinLength != nil && len(str) < *schema.MinLength {
			return fmt.Errorf("%s: length %d is less than minimum %d", path, len(str), *schema.MinLength)
		}
		if schema.MaxLength != nil && len(str) > *schema.MaxLength {
			return fmt.Errorf("%s: length %d exceeds maximum %d", path, len(str), *schema.MaxLength)
		}
		if schema.Pattern != "" {
			re := regexp.MustCompile(schema.Pattern)
			if !re.MatchString(str) {
				return fmt.Errorf("%s: does not match pattern %s", path, schema.Pattern)
			}
		}
	}

	// Number-specific validations
	if num, ok := toFloat64(data); ok {
		if schema.Minimum != nil && num < *schema.Minimum {
			return fmt.Errorf("%s: value %f is less than minimum %f", path, num, *schema.Minimum)
		}
		if schema.Maximum != nil && num > *schema.Maximum {
			return fmt.Errorf("%s: value %f exceeds maximum %f", path, num, *schema.Maximum)
		}
	}

	// Array validation
	if arr, ok := data.([]interface{}); ok {
		if schema.MinItems != nil && len(arr) < *schema.MinItems {
			return fmt.Errorf("%s: item count %d is less than minimum %d", path, len(arr), *schema.MinItems)
		}
		if schema.MaxItems != nil && len(arr) > *schema.MaxItems {
			return fmt.Errorf("%s: item count %d exceeds maximum %d", path, len(arr), *schema.MaxItems)
		}
		if schema.Items != nil {
			for i, item := range arr {
				if err := v.validateValue(item, schema.Items, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
	}

	// Object validation
	if obj, ok := data.(map[string]interface{}); ok {
		// Required fields
		for _, required := range schema.Required {
			if _, exists := obj[required]; !exists {
				return fmt.Errorf("%s: missing required field %s", path, required)
			}
		}
		// Properties validation
		if schema.Properties != nil {
			for propName, propSchema := range schema.Properties {
				if val, exists := obj[propName]; exists {
					if err := v.validateValue(val, propSchema, fmt.Sprintf("%s.%s", path, propName)); err != nil {
						return err
					}
				}
			}
		}
	}

	// Custom validator
	if schema.Type != "" {
		if fn, exists := v.customValidators[schema.Type]; exists {
			if err := fn(data); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
	}

	return nil
}

func (v *Validator) validateType(data interface{}, expectedType string, path string) error {
	switch expectedType {
	case "string":
		_, ok := data.(string)
		if !ok {
			return fmt.Errorf("%s: expected string, got %T", path, data)
		}
	case "number":
		if _, ok := toFloat64(data); !ok {
			return fmt.Errorf("%s: expected number, got %T", path, data)
		}
	case "integer":
		if _, ok := toInt64(data); !ok {
			return fmt.Errorf("%s: expected integer, got %T", path, data)
		}
	case "boolean":
		_, ok := data.(bool)
		if !ok {
			return fmt.Errorf("%s: expected boolean, got %T", path, data)
		}
	case "array":
		_, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("%s: expected array, got %T", path, data)
		}
	case "object":
		_, ok := data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%s: expected object, got %T", path, data)
		}
	}
	return nil
}

func (v *Validator) validateEnum(value interface{}, enum []interface{}, path string) error {
	for _, e := range enum {
		if reflect.DeepEqual(value, e) {
			return nil
		}
	}
	return fmt.Errorf("%s: value %v is not in enum %v", path, value, enum)
}

func (v *Validator) validateString(value interface{}) error {
	_, ok := value.(string)
	if !ok {
		return errors.New("expected string")
	}
	return nil
}

func (v *Validator) validateNumber(value interface{}) error {
	_, ok := toFloat64(value)
	if !ok {
		return errors.New("expected number")
	}
	return nil
}

func (v *Validator) validateInteger(value interface{}) error {
	_, ok := toInt64(value)
	if !ok {
		return errors.New("expected integer")
	}
	return nil
}

func (v *Validator) validateBoolean(value interface{}) error {
	_, ok := value.(bool)
	if !ok {
		return errors.New("expected boolean")
	}
	return nil
}

func (v *Validator) validateArray(value interface{}) error {
	_, ok := value.([]interface{})
	if !ok {
		return errors.New("expected array")
	}
	return nil
}

func (v *Validator) validateObject(value interface{}) error {
	_, ok := value.(map[string]interface{})
	if !ok {
		return errors.New("expected object")
	}
	return nil
}

// ValidateRecommendResult validates RecommendResult against schema.
func (v *Validator) ValidateRecommendResult(result *models.RecommendResult) error {
	if result == nil {
		return errors.New("result is nil")
	}

	schema := GetRecommendResultSchema()
	return v.Validate(result, schema)
}

// Helper functions.
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	}
	return 0, false
}

func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int64:
		return val, true
	case float64:
		if float64(int64(val)) == val {
			return int64(val), true
		}
	case string:
		i, err := strconv.ParseInt(val, 10, 64)
		return i, err == nil
	}
	return 0, false
}

// Validator errors.
var (
	ErrValidationFailed = errors.New("validation failed")
)
