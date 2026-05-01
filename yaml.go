package conf

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	goccyyaml "github.com/goccy/go-yaml"
)

// ParseYaml parses the specified config struct from YAML and environment variables.
// This function applies defaults first, then YAML values, then environment variables.
func ParseYaml(r io.Reader, cfg any) error {
	return parseYamlWithLookup(r, cfg, os.LookupEnv)
}

func parseYamlWithLookup(r io.Reader, cfg any, lookup LookupFunc) error {
	if lookup == nil {
		return errors.New("lookup function is nil")
	}

	fields, err := extractYAMLFields(cfg)
	if err != nil {
		return fmt.Errorf("extract fields from config struct: %w", err)
	}

	if len(fields) == 0 {
		return errors.New("no fields identified in config struct")
	}
	defer cleanupCreatedPointers(fields)

	yamlValues, err := parseYAMLValues(r)
	if err != nil {
		return err
	}

	envValues := getEnvValues(collectFieldsEnvNames(fields), lookup)
	if err := processYAMLFields(fields, yamlValues, envValues); err != nil {
		return err
	}

	return nil
}

func parseYAMLValues(r io.Reader) (map[string]any, error) {
	values := make(map[string]any)
	if err := goccyyaml.NewDecoder(r).Decode(&values); err != nil {
		if errors.Is(err, io.EOF) {
			return values, nil
		}

		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	return values, nil
}

func processYAMLFields(fields []field, yamlValues map[string]any, envValues map[string]string) error {
	for i := range fields {
		fld := &fields[i]

		if err := applyFieldDefault(fld); err != nil {
			return err
		}

		_, yamlOK, err := applyYAMLField(fld, yamlValues)
		if err != nil {
			return err
		}

		value, envKey, envOK := lookupFieldEnv(fld, envValues)
		if envOK {
			if err := applyFieldString(fld, envKey, value); err != nil {
				return err
			}
		}

		if fld.options.required && !yamlOK && !envOK {
			return fmt.Errorf("required field %s (%s) is missing value", fld.name, requiredFieldSources(fld))
		}
	}

	return nil
}

func applyYAMLField(fld *field, yamlValues map[string]any) (string, bool, error) {
	value, key, ok := lookupYAMLField(fld, yamlValues)
	if !ok {
		return key, false, nil
	}

	if err := assignYAMLValue(value, fld.value); err != nil {
		return key, true, fieldAssignmentError(fld, key, yamlValueString(value), err)
	}
	markCreatedPointers(fld)

	return key, true, nil
}

func lookupYAMLField(fld *field, values map[string]any) (any, string, bool) {
	key := yamlPathString(fld)
	var current any = values

	for _, pathPart := range fld.yamlPath {
		currentMap, ok := yamlMap(current)
		if !ok {
			return nil, key, false
		}

		value, ok := currentMap[pathPart]
		if !ok || value == nil {
			return nil, key, false
		}

		current = value
	}

	return current, key, true
}

func yamlMap(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map {
		return nil, false
	}

	values := make(map[string]any, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		key := iter.Key()
		if key.Kind() != reflect.String {
			return nil, false
		}
		values[key.String()] = iter.Value().Interface()
	}

	return values, true
}

func assignYAMLValue(value any, rv reflect.Value) error {
	if value == nil {
		return nil
	}

	typ := rv.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		if rv.IsNil() {
			rv.Set(reflect.New(typ))
		}
		rv = rv.Elem()
	}

	if setterFrom(rv) != nil || textUnmarshaler(rv) != nil || binaryUnmarshaler(rv) != nil {
		scalar, err := yamlScalarString(value)
		if err != nil {
			return err
		}

		return processField(false, scalar, rv)
	}

	switch typ.Kind() {
	case reflect.Slice:
		return assignYAMLSlice(value, rv)
	case reflect.Map:
		return assignYAMLMap(value, rv)
	default:
		scalar, err := yamlScalarString(value)
		if err != nil {
			return err
		}

		return processField(false, scalar, rv)
	}
}

func assignYAMLSlice(value any, rv reflect.Value) error {
	raw := reflect.ValueOf(value)
	if raw.Kind() != reflect.Slice && raw.Kind() != reflect.Array {
		return fmt.Errorf("expected YAML sequence, got %T", value)
	}

	typ := rv.Type()
	result := reflect.MakeSlice(typ, raw.Len(), raw.Len())
	for i := range raw.Len() {
		if err := assignYAMLValue(raw.Index(i).Interface(), result.Index(i)); err != nil {
			return err
		}
	}

	rv.Set(result)

	return nil
}

func assignYAMLMap(value any, rv reflect.Value) error {
	raw := reflect.ValueOf(value)
	if raw.Kind() != reflect.Map {
		return fmt.Errorf("expected YAML map, got %T", value)
	}

	typ := rv.Type()
	result := reflect.MakeMapWithSize(typ, raw.Len())
	iter := raw.MapRange()
	for iter.Next() {
		key := reflect.New(typ.Key()).Elem()
		if err := assignYAMLValue(iter.Key().Interface(), key); err != nil {
			return fmt.Errorf("assign map key: %w", err)
		}

		elem := reflect.New(typ.Elem()).Elem()
		if err := assignYAMLValue(iter.Value().Interface(), elem); err != nil {
			return fmt.Errorf("assign map value for key %s: %w", yamlValueString(iter.Key().Interface()), err)
		}

		result.SetMapIndex(key, elem)
	}

	rv.Set(result)

	return nil
}

func yamlScalarString(value any) (string, error) {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Invalid:
		return "", nil
	case reflect.String:
		return rv.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(rv.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'f', -1, rv.Type().Bits()), nil
	case reflect.Map:
		return "", fmt.Errorf("expected YAML scalar, got %T", value)
	case reflect.Slice, reflect.Array:
		return "", fmt.Errorf("expected YAML scalar, got %T", value)
	default:
		return fmt.Sprint(value), nil
	}
}

func yamlValueString(value any) string {
	scalar, err := yamlScalarString(value)
	if err == nil {
		return scalar
	}

	return fmt.Sprint(value)
}

func yamlPathString(fld *field) string {
	if len(fld.yamlPath) == 0 {
		return strings.ToLower(strings.Join(camelSplit(fld.name), "_"))
	}

	return strings.Join(fld.yamlPath, ".")
}

func requiredFieldSources(fld *field) string {
	envSources := strings.Join(fld.envKeys, " or ")
	if envSources == "" {
		return yamlPathString(fld)
	}

	return fmt.Sprintf("%s or %s", yamlPathString(fld), envSources)
}
