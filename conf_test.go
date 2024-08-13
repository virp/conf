package conf

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const (
	success = "\u2713"
	failed  = "\u2717"
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
func (c CustomValue) String() string {
	return c.something
}

// Equal implements the Equal "interface" for go-cmp
func (c CustomValue) Equal(o CustomValue) bool {
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
			"default",
			nil,
			config{9, "B", false, "", ip{"localhost", "127.0.0.0", []string{"127.0.0.1:200", "127.0.0.1:829"}}, "https://user:password@0.0.0.0:4000", "password", CustomValue{something: "@hello@"}, Embed{"sergey", time.Second}},
		},
		{
			"env",
			map[string]string{"TEST_AN_INT": "1", "TEST_A_STRING": "s", "TEST_BOOL": "TRUE", "TEST_SKIP": "SKIP", "IP_NAME_VAR": "local", "TEST_DEBUG_HOST": "http://sergey:gopher@0.0.0.0:4000", "TEST_PASSWORD": "gopher", "TEST_NAME": "virp", "TEST_DURATION": "1m"},
			config{1, "s", true, "", ip{"local", "127.0.0.0", []string{"127.0.0.1:200", "127.0.0.1:829"}}, "http://sergey:gopher@0.0.0.0:4000", "gopher", CustomValue{something: "@hello@"}, Embed{"virp", time.Minute}},
		},
	}

	for _, tt := range tests {
		os.Clearenv()
		for k, v := range tt.envs {
			_ = os.Setenv(k, v)
		}

		t.Run(tt.name, func(t *testing.T) {
			var cfg config

			if err := Parse("test", &cfg); err != nil {
				t.Fatalf("\t%s\tShould be able to parse environment variables : %s.", failed, err)
			}

			t.Logf("%s\tShould be able to parse environment variables.", success)

			if diff := cmp.Diff(tt.want, cfg); diff != "" {
				t.Fatalf("%s\tShould have properly initialized struct value\n%s", failed, diff)
			}

			t.Logf("%s\tShould have properly initialized struct value.", success)
		})
	}
}

func TestParse_EmptyNamespace(t *testing.T) {
	envs := map[string]string{"AN_INT": "1", "A_STRING": "s", "BOOL": "TRUE", "SKIP": "SKIP", "IP_NAME_VAR": "local", "DEBUG_HOST": "http://bill:gopher@0.0.0.0:4000", "PASSWORD": "gopher", "NAME": "andy", "DURATION": "1m"}
	want := config{1, "s", true, "", ip{"local", "127.0.0.0", []string{"127.0.0.1:200", "127.0.0.1:829"}}, "http://bill:gopher@0.0.0.0:4000", "gopher", CustomValue{something: "@hello@"}, Embed{"andy", time.Minute}}

	os.Clearenv()
	for k, v := range envs {
		_ = os.Setenv(k, v)
	}

	var cfg config

	if err := Parse("", &cfg); err != nil {
		t.Fatalf("%s\tShould be able to parse environment variables : %s.", failed, err)
	}

	t.Logf("%s\tShould be able to parse environment variables.", success)

	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Fatalf("%s\tShould have properly initialized struct value\n%s", failed, diff)
	}

	t.Logf("%s\tShould have properly initialized struct value.", success)
}

func TestParse_Required(t *testing.T) {
	t.Log("When required values are missing.")
	{
		t.Run("required-missing-value", func(t *testing.T) {
			os.Clearenv()

			var cfg struct {
				TestInt    int `conf:"required"`
				TestString string
				TestBool   bool
			}

			err := Parse("test", &cfg)
			if err == nil {
				t.Fatalf("\t%s\tShould fail for missing required value.", failed)
			}

			t.Logf("\t%s\tShould fail for missing required value: %s.", success, err)
		})
	}

	t.Log("When required env integer is zero.")
	{
		t.Run("required-env-integer-zero", func(t *testing.T) {
			os.Clearenv()
			_ = os.Setenv("TEST_TEST_INT", "0")

			var cfg struct {
				TestInt    int `conf:"required"`
				TestString string
				TestBool   bool
			}

			err := Parse("test", &cfg)
			if err != nil {
				t.Fatalf("\t%s\tShould have parsed the required zero env integer : %s.", failed, err)
			}
			t.Logf("\t%s\tShould have parsed the required zero env integer.", success)
		})
	}

	t.Log("When required env string is empty.")
	{
		t.Run("required-env-string-empty", func(t *testing.T) {
			os.Clearenv()
			_ = os.Setenv("TEST_TEST_STRING", "")

			var cfg struct {
				TestInt    int
				TestString string `conf:"required"`
				TestBool   bool
			}

			err := Parse("test", &cfg)
			if err != nil {
				t.Fatalf("\t%s\tShould have parsed the required empty env string : %s.", failed, err)
			}
			t.Logf("\t%s\tShould have parsed the required empty env string.", success)
		})
	}

	t.Log("When required env boolean is false.")
	{
		t.Run("required-env-boolean-false", func(t *testing.T) {
			os.Clearenv()
			_ = os.Setenv("TEST_TEST_BOOL", "false")

			var cfg struct {
				TestInt    int
				TestString string
				TestBool   bool `conf:"required"`
			}

			err := Parse("test", &cfg)
			if err != nil {
				t.Fatalf("\t%s\tSould have parsed the required false env boolean : %s.", failed, err)
			}
			t.Logf("\t%s\tSould have parsed the required false env boolean.", success)
		})
	}

	t.Log("When struct has no fields.")
	{
		t.Run("struct-missing-fields", func(t *testing.T) {
			os.Clearenv()

			var cfg struct {
				testInt    int `conf:"required"`
				testString string
				testBool   bool
			}

			err := Parse("test", &cfg)

			if err == nil {
				t.Fatalf("\t%s\tShould fail for struct with no exported fields.", failed)
			}
			t.Logf("\t%s\tShould fail for struct with no exported fields : %s.", success, err)
		})
	}

	t.Log("When required env value missing error should contain env variable.")
	{
		t.Run("required-env-missing-error-message", func(t *testing.T) {
			os.Clearenv()

			var cfg struct {
				TestInt    int `conf:"required"`
				TestString string
				TestBool   bool
			}

			err := Parse("test", &cfg)
			if err == nil {
				t.Fatalf("\t%s\tShould fail for missing required value.", failed)
			}

			if !strings.Contains(err.Error(), "TEST_TEST_INT") {
				t.Fatalf("\t%s\tShoud fail for missing required value with env variablae name in message : %s.", failed, err)
			}

			t.Logf("\t%s\tShoud fail for missing required value with env variablae name in message : %s.", success, err)
		})
	}
}

func TestParse_Errors(t *testing.T) {
	t.Log("When passing struct to Parse.")
	{
		t.Run("not-by-ref", func(t *testing.T) {
			var cfg struct {
				TestInt    int
				TestString string
				TestBool   bool
			}

			err := Parse("test", cfg)
			if err == nil {
				t.Fatalf("\t%s\tShould NOT be able to accept a value by value.", failed)
			}
			t.Logf("\t%s\tShould NOT be able to accept a value by value : %s.", success, err)
		})

		t.Run("not-struct-value", func(t *testing.T) {
			var cfg []string

			err := Parse("test", &cfg)
			if err == nil {
				t.Fatalf("\t%s\tShould NOT be able to pass anything but a struct value.", failed)
			}
			t.Logf("\t%s\tShould NOT be able to pass anything but a struct value : %s.", success, err)
		})
	}

	t.Log("When bad tags to Parse.")
	{
		t.Run("tag-missing-value", func(t *testing.T) {
			var cfg struct {
				TestInt    int `conf:"default:"`
				TestString string
				TestBool   bool
			}

			err := Parse("test", &cfg)
			if err == nil {
				t.Fatalf("\t%s\tShould NOT be able to accept tag missing value.", failed)
			}
			t.Logf("\t%s\tShould NOT be able to accept tag missing value : %s.", success, err)
		})
	}
}
