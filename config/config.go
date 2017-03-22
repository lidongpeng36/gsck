package config

import (
	"os"
	"path"
	"strings"

	// "gopkg.in/ini.v1"
	"github.com/go-ini/ini"
)

type Setting struct {
	Path      string
	FileName  string
	EnvPrefix string
	Defaults  map[string]string
}

const defaultSection = "default"

var configPath string
var conf *ini.File

func Setup(s *Setting) {
	configPath = path.Join(s.Path, s.FileName)
	_, e := os.Lstat(configPath)
	if os.IsNotExist(e) {
		conf = ini.Empty()
		_ = conf.SaveTo(configPath)
	} else {
		conf, _ = ini.Load(configPath)
	}
	if s.Defaults != nil {
		SetDefaultFromMap(s.Defaults)
	}
	bindEnv(s.EnvPrefix)
}

func setDefault(raw, value string) {
	section, key := splitKey(raw)
	if "" == section.Key(key).String() {
		_, _ = section.NewKey(key, value)
	}
}

// SetDefaultFromMap could be used to override defaults by plugins
func SetDefaultFromMap(pair map[string]string) {
	for k, v := range pair {
		setDefault(k, v)
	}
	_ = conf.SaveTo(configPath)
}

// bindEnv sets ENV `prefix`_X with value of default.X
// Actual effect, if written in bash: export prefix_VAR=default.VAR
func bindEnv(prefix string) {
	section := conf.Section(defaultSection)
	hash := section.KeysHash()
	for k, v := range hash {
		envKey := strings.ToUpper(k)
		if "" != prefix {
			envKey = prefix + "_" + envKey
		}
		if "" == os.Getenv(envKey) {
			_ = os.Setenv(envKey, v)

		}
	}
}

func splitKey(raw string) (section *ini.Section, key string) {
	fields := strings.Split(raw, ".")
	length := len(fields)
	sectionName := defaultSection
	if 1 < length {
		sectionName = fields[0]
	}
	section = conf.Section(sectionName)
	key = fields[length-1]
	return
}

// GetString returns string value for key
func GetString(key string) (value string) {
	section, sectionKey := splitKey(key)
	value = strings.TrimSpace(section.Key(sectionKey).String())
	return
}

// GetInt returns int value for key
func GetInt(key string) (value int) {
	section, sectionKey := splitKey(key)
	value, _ = section.Key(sectionKey).Int()
	return
}

// GetBool returns bool value for key
func GetBool(key string) (value bool) {
	section, sectionKey := splitKey(key)
	value, _ = section.Key(sectionKey).Bool()
	return
}

// Set writes setting to config file
func Set(key, value string) {
	section, sectionKey := splitKey(key)
	_, _ = section.NewKey(sectionKey, value)
	_ = conf.SaveTo(configPath)
}
