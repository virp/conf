package conf

import (
	"errors"
	"fmt"
	"os"
)

// Parse parses the specified config struct.
// This function will apply the defaults first and then
// apply environment variables to the struct.
func Parse(prefix string, cfg any) error {

	// Get the list of fields from the configuration struct to process.
	fields, err := extractFields(prefix, cfg)
	if err != nil {
		return fmt.Errorf("extract fields from config struct: %w", err)
	}

	if len(fields) == 0 {
		return errors.New("no fields identified in config struct")
	}

	// Collect all env names for fields.
	envNames := collectFieldsEnvNames(fields)

	// Get all existed env variables values for fields.
	envValues := getEnvValues(envNames)

	// Process all fields found in the config struct provided.
	if err := processFields(fields, envValues); err != nil {
		return err
	}

	return nil
}

func collectFieldsEnvNames(fields []Field) []string {
	envNames := make([]string, 0, len(fields))

	for _, field := range fields {
		envNames = append(envNames, field.EnvKey)
	}

	return envNames
}

func getEnvValues(envNames []string) map[string]string {
	envValues := make(map[string]string)

	for _, envName := range envNames {
		if value, ok := os.LookupEnv(envName); ok {
			envValues[envName] = value
		}
	}

	return envValues
}

func processFields(fields []Field, envValues map[string]string) error {
	for _, field := range fields {

		// Set any default value into the struct for this field.
		if field.Options.DefaultVal != "" {
			if err := processField(true, field.Options.DefaultVal, field.Field); err != nil {
				return &FieldError{
					fieldName: field.Name,
					envKey:    field.EnvKey,
					typeName:  field.Field.Type().String(),
					value:     field.Options.DefaultVal,
					err:       err,
				}
			}
		}

		value, ok := envValues[field.EnvKey]

		if field.Options.Required && !ok {
			return fmt.Errorf("required field %s (%s) is missing value", field.Name, field.EnvKey)
		}

		if !ok {
			continue
		}

		// A value was found so update the struct value with it.
		if err := processField(false, value, field.Field); err != nil {
			return &FieldError{
				fieldName: field.Name,
				envKey:    field.EnvKey,
				typeName:  field.Field.Type().String(),
				value:     field.Options.DefaultVal,
				err:       err,
			}
		}
	}

	return nil
}
