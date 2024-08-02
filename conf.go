package conf

import (
	"errors"
	"fmt"
)

// Parse parses the specified config struct.
// This function will apply the defaults first and then
// apply environment variables to the struct.
func Parse(prefix string, cfg any) error {

	// Get the list of fields from the configuration struct to process.
	fields, err := extractFields(nil, cfg)
	if err != nil {
		return err
	}

	if len(fields) == 0 {
		return errors.New("no fields identified in config struct")
	}

	var specifiedEnvNames []string
	for _, field := range fields {
		if field.Options.EnvName != "" {
			specifiedEnvNames = append(specifiedEnvNames, field.Options.EnvName)
		}
	}

	// Get the env variables.
	env := newEnv(prefix, specifiedEnvNames)

	// Process all fields found in the config struct provided.
	for _, field := range fields {
		field := field

		// Set any default value into the struct for this field.
		if field.Options.DefaultVal != "" {
			if err := processField(true, field.Options.DefaultVal, field.Field); err != nil {
				return &FieldError{
					fieldName: field.Name,
					typeName:  field.Field.Type().String(),
					value:     field.Options.DefaultVal,
					err:       err,
				}
			}
		}

		value, ok := env.Value(field)

		// If filed is marked 'required', check if no value was provided.
		if field.Options.Required && !ok {
			return fmt.Errorf("required field %s is missing value", field.Name)
		}

		if !ok {
			continue
		}

		// A value was found so update the struct value with it.
		if err := processField(false, value, field.Field); err != nil {
			return &FieldError{
				fieldName: field.Name,
				typeName:  field.Field.Type().String(),
				value:     value,
				err:       err,
			}
		}
	}

	return nil
}
