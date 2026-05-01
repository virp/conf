package conf

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseYaml(t *testing.T) {
	setEnvs(t, map[string]string{
		"AN_INT":      "42",
		"IP_NAME_VAR": "env-ip",
		"NAME":        "env-embed",
	})

	data := strings.NewReader(`
an_int: 11
a_string: yaml
bool: true
ip:
  name: yaml-ip
  ip: 10.0.0.1
  endpoints:
    - 10.0.0.1:200
    - 10.0.0.2:829
debug_host: https://yaml.example.com
password: yaml-password
custom: yaml-custom
name: yaml-embed
duration: 2s
`)

	var cfg config
	require.NoError(t, ParseYaml(data, &cfg))

	assert.Equal(t, config{
		AnInt:     42,
		AString:   "yaml",
		Bool:      true,
		IP:        ip{Name: "env-ip", IP: "10.0.0.1", Endpoints: []string{"10.0.0.1:200", "10.0.0.2:829"}},
		DebugHost: "https://yaml.example.com",
		Password:  "yaml-password",
		Custom:    CustomValue{something: "@yaml-custom@"},
		Embed:     Embed{Name: "env-embed", Duration: 2 * time.Second},
	}, cfg)
}

func TestParseYamlWithLookup(t *testing.T) {
	data := strings.NewReader(`
an_int: 11
ip:
  name: yaml-ip
name: yaml-embed
`)

	var cfg config
	err := parseYamlWithLookup(data, &cfg, mapLookup(map[string]string{
		"AN_INT":      "42",
		"IP_NAME_VAR": "env-ip",
	}))

	require.NoError(t, err)
	assert.Equal(t, 42, cfg.AnInt)
	assert.Equal(t, "env-ip", cfg.IP.Name)
	assert.Equal(t, "yaml-embed", cfg.Name)
}

func TestParseYaml_YAMLTags(t *testing.T) {
	data := strings.NewReader(`
http_port: 8080
custom_name: yaml
hidden: visible
skip: visible
`)

	var cfg struct {
		HTTPPort int
		Renamed  string `yaml:"custom_name"`
		Hidden   string `yaml:"-"`
		Skip     string `conf:"-"`
	}

	require.NoError(t, parseYamlWithLookup(data, &cfg, mapLookup(nil)))
	assert.Equal(t, 8080, cfg.HTTPPort)
	assert.Equal(t, "yaml", cfg.Renamed)
	assert.Empty(t, cfg.Hidden)
	assert.Empty(t, cfg.Skip)
}

func TestParseYaml_Required(t *testing.T) {
	t.Run("yaml-value", func(t *testing.T) {
		var cfg struct {
			Name string `conf:"required"`
		}

		require.NoError(t, parseYamlWithLookup(strings.NewReader("name: yaml"), &cfg, mapLookup(nil)))
		assert.Equal(t, "yaml", cfg.Name)
	})

	t.Run("env-value", func(t *testing.T) {
		var cfg struct {
			Name string `conf:"required"`
		}

		require.NoError(t, parseYamlWithLookup(strings.NewReader(""), &cfg, mapLookup(map[string]string{"NAME": "env"})))
		assert.Equal(t, "env", cfg.Name)
	})

	t.Run("missing-value", func(t *testing.T) {
		var cfg struct {
			Name string `conf:"required"`
		}

		err := parseYamlWithLookup(strings.NewReader(""), &cfg, mapLookup(nil))
		require.Error(t, err)
		require.ErrorContains(t, err, "name")
		assert.ErrorContains(t, err, "NAME")
	})

	t.Run("null-is-missing", func(t *testing.T) {
		var cfg struct {
			Name string `conf:"default:default"`
			Port int    `conf:"required"`
		}

		err := parseYamlWithLookup(strings.NewReader("name: null\nport: null"), &cfg, mapLookup(nil))
		require.Error(t, err)
		assert.Equal(t, "default", cfg.Name)
	})
}

func TestParseYaml_NativeSlicesAndMaps(t *testing.T) {
	data := strings.NewReader(`
endpoints:
  - localhost:6379
  - localhost:6380
ports:
  - 80
  - 443
routes:
  api: https://api.example.com:443/path
  admin: http://localhost:8080
weights:
  10: 1.5
  20: 2.5
`)

	var cfg struct {
		Endpoints []string
		Ports     []int
		Routes    map[string]string
		Weights   map[int]float64
	}

	require.NoError(t, parseYamlWithLookup(data, &cfg, mapLookup(nil)))
	assert.Equal(t, []string{"localhost:6379", "localhost:6380"}, cfg.Endpoints)
	assert.Equal(t, []int{80, 443}, cfg.Ports)
	assert.Equal(t, map[string]string{
		"api":   "https://api.example.com:443/path",
		"admin": "http://localhost:8080",
	}, cfg.Routes)
	assert.Equal(t, map[int]float64{10: 1.5, 20: 2.5}, cfg.Weights)
}

func TestParseYaml_NilStructPointer(t *testing.T) {
	t.Run("keeps-unused-pointer-nil", func(t *testing.T) {
		var cfg struct {
			Nested *struct {
				Value string
			}
		}

		require.NoError(t, parseYamlWithLookup(strings.NewReader(""), &cfg, mapLookup(nil)))
		assert.Nil(t, cfg.Nested)
	})

	t.Run("initializes-pointer-for-yaml-value", func(t *testing.T) {
		var cfg struct {
			Nested *struct {
				Value string
			}
		}

		require.NoError(t, parseYamlWithLookup(strings.NewReader("nested:\n  value: from-yaml"), &cfg, mapLookup(nil)))
		require.NotNil(t, cfg.Nested)
		assert.Equal(t, "from-yaml", cfg.Nested.Value)
	})

	t.Run("keeps-null-pointer-nil", func(t *testing.T) {
		var cfg struct {
			Nested *struct {
				Value string
			}
		}

		require.NoError(t, parseYamlWithLookup(strings.NewReader("nested: null"), &cfg, mapLookup(nil)))
		assert.Nil(t, cfg.Nested)
	})
}

func TestParseYaml_ScalarConversions(t *testing.T) {
	data := strings.NewReader(`
custom: hello
duration: 3s
count: 10
ratio: 1.5
flag: true
`)

	var cfg struct {
		Custom   CustomValue
		Duration time.Duration
		Count    int
		Ratio    float64
		Flag     bool
	}

	require.NoError(t, parseYamlWithLookup(data, &cfg, mapLookup(nil)))
	assert.Equal(t, CustomValue{something: "@hello@"}, cfg.Custom)
	assert.Equal(t, 3*time.Second, cfg.Duration)
	assert.Equal(t, 10, cfg.Count)
	assert.InDelta(t, 1.5, cfg.Ratio, 0.001)
	assert.True(t, cfg.Flag)
}

func TestParseYaml_Errors(t *testing.T) {
	t.Run("invalid-yaml", func(t *testing.T) {
		var cfg config
		err := parseYamlWithLookup(strings.NewReader("["), &cfg, mapLookup(nil))
		require.Error(t, err)
		assert.ErrorContains(t, err, "parse yaml")
	})

	t.Run("invalid-conversion", func(t *testing.T) {
		var cfg struct {
			AnInt int
		}

		err := parseYamlWithLookup(strings.NewReader("an_int: not-an-int"), &cfg, mapLookup(nil))
		require.Error(t, err)
		require.ErrorContains(t, err, "an_int")
		assert.ErrorContains(t, err, "not-an-int")
	})

	t.Run("nil-lookup", func(t *testing.T) {
		var cfg config
		err := parseYamlWithLookup(strings.NewReader(""), &cfg, nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "lookup function is nil")
	})
}
