package conf

import (
	"fmt"
	"os"
	"strings"
)

// env is a source for environment variables.
type env struct {
	m map[string]string
}

// Value returns the string value stored at the specified key from the environment.
func (e *env) Value(fld Field) (string, bool) {
	k := strings.ToUpper(strings.Join(fld.EnvKeys, "_"))
	v, ok := e.m[k]

	return v, ok
}

// newEnv accepts a prefix as a namespace and parses environment variables into a env.
func newEnv(prefix string, include []string) *env {
	m := make(map[string]string)

	includeMap := make(map[string]struct{}, len(include))
	for _, n := range include {
		includeMap[n] = struct{}{}
	}

	// Create the uppercase version to meet the standard {NAMESPACE_} format.
	// If the namespace is empty, remote _ from the beginning of the string.
	namespace := fmt.Sprintf("%s_", strings.ToUpper(prefix))
	if prefix == "" {
		namespace = namespace[1:]
	}

	// Loop and match each variable using the uppercase namespace.
	for _, val := range os.Environ() {
		idx := strings.Index(val, "=")

		_, givenEnvName := includeMap[val[0:idx]]
		if !strings.HasPrefix(val, namespace) && !givenEnvName {
			continue
		}

		m[strings.ToUpper(strings.TrimPrefix(val[0:idx], namespace))] = val[idx+1:]
	}

	return &env{m: m}
}
