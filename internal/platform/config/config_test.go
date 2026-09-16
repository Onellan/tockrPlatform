package config

import (
	"strings"
	"testing"
)

func TestFromEnvironmentUsesSafeDefaultsAndCopiesKey(t *testing.T) {
	values := map[string]string{"PLATFORM_MFA_KEY": strings.Repeat("ab", 32)}
	config, err := FromEnvironment(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if config.DBPath != "platform.db" || config.HTTPAddr != ":8080" || config.AllowInsecureCookie {
		t.Fatalf("defaults = %#v", config)
	}
	if len(config.MFAKey) != 32 {
		t.Fatalf("MFA key length = %d", len(config.MFAKey))
	}
	values["PLATFORM_MFA_KEY"] = strings.Repeat("cd", 32)
	if config.MFAKey[0] != 0xab {
		t.Fatal("MFA key was not copied from the environment")
	}
}

func TestFromEnvironmentRejectsInvalidConfigurationWithoutSecretDisclosure(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
	}{
		{name: "missing key", values: map[string]string{}},
		{name: "invalid key", values: map[string]string{"PLATFORM_MFA_KEY": "not-a-key"}},
		{name: "invalid address", values: map[string]string{"PLATFORM_MFA_KEY": strings.Repeat("ab", 32), "PLATFORM_HTTP_ADDR": "http://localhost:8080"}},
		{name: "invalid boolean", values: map[string]string{"PLATFORM_MFA_KEY": strings.Repeat("ab", 32), "PLATFORM_ALLOW_INSECURE_COOKIES": "yes"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := FromEnvironment(func(key string) string { return test.values[key] })
			if err == nil || !strings.Contains(err.Error(), ErrConfiguration.Error()) {
				t.Fatalf("error = %v", err)
			}
			if strings.Contains(err.Error(), "abab") || strings.Contains(err.Error(), "cdcd") {
				t.Fatalf("error disclosed secret material: %v", err)
			}
		})
	}
}

func TestFromEnvironmentAcceptsExplicitRuntimeOptions(t *testing.T) {
	values := map[string]string{
		"PLATFORM_MFA_KEY":                strings.Repeat("ab", 32),
		"PLATFORM_DB_PATH":                "/var/lib/tockrplatform/platform.db",
		"PLATFORM_HTTP_ADDR":              "127.0.0.1:9090",
		"PLATFORM_ALLOW_INSECURE_COOKIES": "1",
	}
	config, err := FromEnvironment(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if config.DBPath != values["PLATFORM_DB_PATH"] || config.HTTPAddr != values["PLATFORM_HTTP_ADDR"] || !config.AllowInsecureCookie {
		t.Fatalf("explicit configuration = %#v", config)
	}
}
