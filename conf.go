package conf

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Parse parses the specified config struct.
// This function will apply the defaults first and then
// apply environment variables to the struct.
func Parse(prefix string, cfg any) error {
	return ParseWithLookup(prefix, cfg, os.LookupEnv)
}

// LookupFunc looks up a value by environment variable name.
type LookupFunc func(key string) (string, bool)

// ParseWithLookup parses the specified config struct using lookup as the
// environment variable source.
func ParseWithLookup(prefix string, cfg any, lookup LookupFunc) error {
	if lookup == nil {
		return errors.New("lookup function is nil")
	}

	// Get the list of fields from the configuration struct to process.
	fields, err := extractFields(prefix, cfg)
	if err != nil {
		return fmt.Errorf("extract fields from config struct: %w", err)
	}

	if len(fields) == 0 {
		return errors.New("no fields identified in config struct")
	}
	defer cleanupCreatedPointers(fields)

	// Collect all env names for fields.
	envNames := collectFieldsEnvNames(fields)

	// Get all existed env variables values for fields.
	envValues := getEnvValues(envNames, lookup)

	// Process all fields found in the config struct provided.
	if err := processFields(fields, envValues); err != nil {
		return err
	}

	return nil
}

func collectFieldsEnvNames(fields []field) []string {
	envNames := make([]string, 0, len(fields))

	for i := range fields {
		envNames = append(envNames, fields[i].envKeys...)
	}

	return envNames
}

func getEnvValues(envNames []string, lookup LookupFunc) map[string]string {
	envValues := make(map[string]string)

	for _, envName := range envNames {
		if value, ok := lookup(envName); ok {
			envValues[envName] = value
		}
	}

	return envValues
}

func processFields(fields []field, envValues map[string]string) error {
	for i := range fields {
		fld := &fields[i]

		// Set any default value into the struct for this field.
		if fld.options.defaultVal != "" {
			if err := processField(true, fld.options.defaultVal, fld.value); err != nil {
				return &FieldError{
					fieldName: fld.name,
					envKey:    primaryEnvKey(fld),
					typeName:  fld.value.Type().String(),
					value:     fld.options.defaultVal,
					err:       err,
				}
			}
			markCreatedPointers(fld)
		}

		value, envKey, ok := lookupFieldEnv(fld, envValues)

		if fld.options.required && !ok {
			return fmt.Errorf("required field %s (%s) is missing value", fld.name, strings.Join(fld.envKeys, " or "))
		}

		if !ok {
			continue
		}

		// A value was found so update the struct value with it.
		if err := processField(false, value, fld.value); err != nil {
			return &FieldError{
				fieldName: fld.name,
				envKey:    envKey,
				typeName:  fld.value.Type().String(),
				value:     value,
				err:       err,
			}
		}
		markCreatedPointers(fld)
	}

	return nil
}

func primaryEnvKey(fld *field) string {
	if len(fld.envKeys) == 0 {
		return ""
	}

	return fld.envKeys[0]
}

func markCreatedPointers(fld *field) {
	for _, createdPtr := range fld.createdPtrs {
		createdPtr.touched = true
	}
}

func cleanupCreatedPointers(fields []field) {
	var createdPtrs []*createdPointer
	seen := make(map[*createdPointer]bool)

	for i := range fields {
		for _, createdPtr := range fields[i].createdPtrs {
			if seen[createdPtr] {
				continue
			}

			seen[createdPtr] = true
			createdPtrs = append(createdPtrs, createdPtr)
		}
	}

	cleanupCreatedPointerList(createdPtrs)
}

func lookupFieldEnv(fld *field, envValues map[string]string) (string, string, bool) {
	for _, envKey := range fld.envKeys {
		if value, ok := envValues[envKey]; ok {
			return value, envKey, true
		}
	}

	return "", "", false
}
