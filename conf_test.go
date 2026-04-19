package conf

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CustomValue provides support for testing a custom value.
type CustomValue struct {
	something string
}

// Set implements the Setter interface
func (c *CustomValue) Set(data string) error {
	*c = CustomValue{something: fmt.Sprintf("@%s@", data)}
	return nil
}

// String implements the Stringer interface
func (c *CustomValue) String() string {
	return c.something
}

// Equal compares custom values in tests.
func (c *CustomValue) Equal(o *CustomValue) bool {
	return c.something == o.something
}

type ip struct {
	Name      string   `conf:"default:localhost,env:IP_NAME_VAR"`
	IP        string   `conf:"default:127.0.0.0"`
	Endpoints []string `conf:"default:127.0.0.1:200;127.0.0.1:829"`
}

type Embed struct {
	Name     string        `conf:"default:sergey"`
	Duration time.Duration `conf:"default:1s"`
}

type config struct {
	AnInt     int    `conf:"default:9"`
	AString   string `conf:"default:B"`
	Bool      bool
	Skip      string `conf:"-"`
	IP        ip
	DebugHost string      `conf:"default:https://user:password@0.0.0.0:4000"`
	Password  string      `conf:"default:password"`
	Custom    CustomValue `conf:"default:hello"`
	Embed
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		envs map[string]string
		want config
	}{
		{
			name: "default",
			want: config{
				AnInt:     9,
				AString:   "B",
				IP:        ip{Name: "localhost", IP: "127.0.0.0", Endpoints: []string{"127.0.0.1:200", "127.0.0.1:829"}},
				DebugHost: "https://user:password@0.0.0.0:4000",
				Password:  "password",
				Custom:    CustomValue{something: "@hello@"},
				Embed:     Embed{Name: "sergey", Duration: time.Second},
			},
		},
		{
			name: "env",
			envs: map[string]string{
				"TEST_AN_INT":     "1",
				"TEST_A_STRING":   "s",
				"TEST_BOOL":       "TRUE",
				"TEST_SKIP":       "SKIP",
				"IP_NAME_VAR":     "local",
				"TEST_DEBUG_HOST": "http://sergey:gopher@0.0.0.0:4000",
				"TEST_PASSWORD":   "gopher",
				"TEST_NAME":       "virp",
				"TEST_DURATION":   "1m",
			},
			want: config{
				AnInt:     1,
				AString:   "s",
				Bool:      true,
				IP:        ip{Name: "local", IP: "127.0.0.0", Endpoints: []string{"127.0.0.1:200", "127.0.0.1:829"}},
				DebugHost: "http://sergey:gopher@0.0.0.0:4000",
				Password:  "gopher",
				Custom:    CustomValue{something: "@hello@"},
				Embed:     Embed{Name: "virp", Duration: time.Minute},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvs(t, tt.envs)

			var cfg config
			require.NoError(t, Parse("test", &cfg))
			assert.Equal(t, tt.want, cfg)
		})
	}
}

func TestParse_EmptyPrefix(t *testing.T) {
	setEnvs(t, map[string]string{
		"AN_INT":      "1",
		"A_STRING":    "s",
		"BOOL":        "TRUE",
		"SKIP":        "SKIP",
		"IP_NAME_VAR": "local",
		"DEBUG_HOST":  "http://bill:gopher@0.0.0.0:4000",
		"PASSWORD":    "gopher",
		"NAME":        "andy",
		"DURATION":    "1m",
	})

	want := config{
		AnInt:     1,
		AString:   "s",
		Bool:      true,
		IP:        ip{Name: "local", IP: "127.0.0.0", Endpoints: []string{"127.0.0.1:200", "127.0.0.1:829"}},
		DebugHost: "http://bill:gopher@0.0.0.0:4000",
		Password:  "gopher",
		Custom:    CustomValue{something: "@hello@"},
		Embed:     Embed{Name: "andy", Duration: time.Minute},
	}

	var cfg config
	require.NoError(t, Parse("", &cfg))
	assert.Equal(t, want, cfg)
}

func TestParseWithLookup(t *testing.T) {
	tests := []struct {
		name        string
		envs        map[string]string
		processEnv  map[string]string
		lookup      LookupFunc
		nilLookup   bool
		requireFunc func(*testing.T, config, error)
	}{
		{
			name: "uses-injected-lookup",
			envs: map[string]string{
				"TEST_AN_INT":     "42",
				"TEST_A_STRING":   "lookup",
				"TEST_BOOL":       "true",
				"TEST_DEBUG_HOST": "http://lookup.example.com",
				"TEST_NAME":       "lookup-name",
				"TEST_DURATION":   "2m",
			},
			requireFunc: func(t *testing.T, cfg config, err error) {
				require.NoError(t, err)
				assert.Equal(t, config{
					AnInt:     42,
					AString:   "lookup",
					Bool:      true,
					IP:        ip{Name: "localhost", IP: "127.0.0.0", Endpoints: []string{"127.0.0.1:200", "127.0.0.1:829"}},
					DebugHost: "http://lookup.example.com",
					Password:  "password",
					Custom:    CustomValue{something: "@hello@"},
					Embed:     Embed{Name: "lookup-name", Duration: 2 * time.Minute},
				}, cfg)
			},
		},
		{
			name:       "does-not-read-process-env",
			processEnv: map[string]string{"TEST_AN_INT": "99"},
			envs:       nil,
			requireFunc: func(t *testing.T, cfg config, err error) {
				require.NoError(t, err)
				assert.Equal(t, 9, cfg.AnInt)
			},
		},
		{
			name:      "nil-lookup",
			nilLookup: true,
			requireFunc: func(t *testing.T, _ config, err error) {
				assert.ErrorContains(t, err, "lookup function is nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvs(t, tt.processEnv)

			lookup := tt.lookup
			if lookup == nil && !tt.nilLookup {
				lookup = mapLookup(tt.envs)
			}

			var cfg config
			err := ParseWithLookup("test", &cfg, lookup)
			tt.requireFunc(t, cfg, err)
		})
	}
}

func mapLookup(values map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestFieldEnvKey(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		fieldName string
		want      string
	}{
		{
			name:      "with-prefix",
			prefix:    "my_service",
			fieldName: "DebugHost",
			want:      "MY_SERVICE_DEBUG_HOST",
		},
		{
			name:      "empty-prefix",
			prefix:    "",
			fieldName: "DebugHost",
			want:      "DEBUG_HOST",
		},
		{
			name:      "acronym",
			prefix:    "my_service",
			fieldName: "HTTPPort",
			want:      "MY_SERVICE_HTTP_PORT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fieldEnvKey(tt.prefix, tt.fieldName))
		})
	}
}

func TestParse_EnvNameFallback(t *testing.T) {
	tests := []struct {
		name string
		envs map[string]string
		want string
	}{
		{
			name: "explicit-env-has-priority",
			envs: map[string]string{
				"IP_NAME_VAR":  "explicit",
				"TEST_IP_NAME": "generated",
			},
			want: "explicit",
		},
		{
			name: "generated-env-is-fallback",
			envs: map[string]string{"TEST_IP_NAME": "generated"},
			want: "generated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvs(t, tt.envs)

			var cfg config
			require.NoError(t, Parse("test", &cfg))
			assert.Equal(t, tt.want, cfg.IP.Name)
		})
	}
}

func TestParse_MapValueWithColon(t *testing.T) {
	setEnvs(t, map[string]string{
		"TEST_ROUTES": "callback:https://example.com:443/path;health:http://localhost:8080/health",
	})

	var cfg struct {
		Routes map[string]string
	}

	require.NoError(t, Parse("test", &cfg))
	assert.Equal(t, map[string]string{
		"callback": "https://example.com:443/path",
		"health":   "http://localhost:8080/health",
	}, cfg.Routes)
}

func TestParse_NilStructPointer(t *testing.T) {
	t.Run("keeps-unused-pointer-nil", func(t *testing.T) {
		setEnvs(t, nil)

		var cfg struct {
			Nested *struct {
				Value string
			}
		}

		require.NoError(t, Parse("test", &cfg))
		assert.Nil(t, cfg.Nested)
	})

	t.Run("initializes-pointer-for-env-value", func(t *testing.T) {
		setEnvs(t, map[string]string{"TEST_NESTED_VALUE": "from-env"})

		var cfg struct {
			Nested *struct {
				Value string
			}
		}

		require.NoError(t, Parse("test", &cfg))
		require.NotNil(t, cfg.Nested)
		assert.Equal(t, "from-env", cfg.Nested.Value)
	})

	t.Run("initializes-pointer-for-default-value", func(t *testing.T) {
		setEnvs(t, nil)

		var cfg struct {
			Nested *struct {
				Value string `conf:"default:from-default"`
			}
		}

		require.NoError(t, Parse("test", &cfg))
		require.NotNil(t, cfg.Nested)
		assert.Equal(t, "from-default", cfg.Nested.Value)
	})
}

func TestParse_Required(t *testing.T) {
	t.Run("required-missing-value", func(t *testing.T) {
		setEnvs(t, nil)

		var cfg struct {
			TestInt    int `conf:"required"`
			TestString string
			TestBool   bool
		}

		assert.Error(t, Parse("test", &cfg))
	})

	requiredValueTests := []struct {
		name string
		envs map[string]string
		cfg  any
	}{
		{
			name: "required-env-integer-zero",
			envs: map[string]string{"TEST_TEST_INT": "0"},
			cfg: &struct {
				TestInt    int `conf:"required"`
				TestString string
				TestBool   bool
			}{},
		},
		{
			name: "required-env-string-empty",
			envs: map[string]string{"TEST_TEST_STRING": ""},
			cfg: &struct {
				TestInt    int
				TestString string `conf:"required"`
				TestBool   bool
			}{},
		},
		{
			name: "required-env-boolean-false",
			envs: map[string]string{"TEST_TEST_BOOL": "false"},
			cfg: &struct {
				TestInt    int
				TestString string
				TestBool   bool `conf:"required"`
			}{},
		},
	}

	for _, tt := range requiredValueTests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvs(t, tt.envs)
			assert.NoError(t, Parse("test", tt.cfg))
		})
	}

	errorTests := []struct {
		name            string
		cfg             any
		wantErrContains string
	}{
		{
			name: "struct-missing-fields",
			cfg: &struct {
				testInt    int `conf:"required"`
				testString string
				testBool   bool
			}{},
		},
		{
			name: "required-env-missing-error-message",
			cfg: &struct {
				TestInt    int `conf:"required"`
				TestString string
				TestBool   bool
			}{},
			wantErrContains: "TEST_TEST_INT",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvs(t, nil)
			err := Parse("test", tt.cfg)
			require.Error(t, err)
			if tt.wantErrContains != "" {
				assert.ErrorContains(t, err, tt.wantErrContains)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	tests := []struct {
		name               string
		envs               map[string]string
		cfg                any
		wantErrContains    []string
		wantErrNotContains []string
	}{
		{
			name: "not-by-ref",
			cfg: struct {
				TestInt    int
				TestString string
				TestBool   bool
			}{},
		},
		{
			name: "not-struct-value",
			cfg:  &[]string{},
		},
		{
			name: "tag-missing-value",
			cfg: &struct {
				TestInt    int `conf:"default:"`
				TestString string
				TestBool   bool
			}{},
		},
		{
			name: "unknown-tag",
			cfg: &struct {
				TestInt int `conf:"requiredx"`
			}{},
			wantErrContains: []string{`unknown tag "requiredx"`},
		},
		{
			name: "unknown-tag-with-value",
			cfg: &struct {
				TestInt int `conf:"source:TEST_TEST_INT"`
			}{},
			wantErrContains: []string{`unknown tag "source"`},
		},
		{
			name: "field-error-contains-env-value",
			envs: map[string]string{"TEST_TEST_INT": "not-an-int"},
			cfg: &struct {
				TestInt int `conf:"default:10"`
			}{},
			wantErrContains:    []string{"not-an-int"},
			wantErrNotContains: []string{"converting '10'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvs(t, tt.envs)

			err := Parse("test", tt.cfg)
			require.Error(t, err)
			for _, want := range tt.wantErrContains {
				require.ErrorContains(t, err, want)
			}
			for _, notWant := range tt.wantErrNotContains {
				assert.NotContains(t, fmt.Sprint(err), notWant)
			}
		})
	}
}

func setEnvs(t *testing.T, envs map[string]string) {
	t.Helper()

	original := os.Environ()
	t.Cleanup(func() {
		os.Clearenv()
		for _, entry := range original {
			key, value, _ := strings.Cut(entry, "=")
			require.NoError(t, os.Setenv(key, value))
		}
	})

	os.Clearenv()
	for key, value := range envs {
		require.NoError(t, os.Setenv(key, value))
	}
}
