/*
The MIT License

Copyright (c) 2018 Maurizio Tomasi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package qutedb

import (
	"encoding/base64"
	"fmt"
	"runtime"

	"github.com/spf13/viper"
)

// A Configuration type contains all the settings loaded from environment
// variables and configuration files. It is initialized using the Viper library.
type Configuration struct {
	// Full path of the file containing the configuration
	ConfigurationFileName string `json:"-"`

	DatabaseFile string `json:"database_file"`

	LogFormat string `json:"log_format"`
	LogLevel  string `json:"log_level"`
	LogOutput string `json:"log_output"`

	PortNumber int    `json:"port_number"`
	ServerName string `json:"server_name"`

	ReadTimeout  int64 `json:"read_timeout"`
	WriteTimeout int64 `json:"write_timeout"`

	StaticPath string `json:"static_path"`

	RepositoryPath string `json:"repository_path"`

	CookieHashKey  []byte `json:"cookie_hash_key"`
	CookieBlockKey []byte `json:"cookie_block_key"`
}

// configureViper sets up the Viper library so that it can read the
// configuration file from a variety of locations.
func configureViper() {
	// Set a number of default values for configuration parameters
	viper.SetDefault("database_file", "db.sqlite3")
	viper.SetDefault("log_format", "text")
	viper.SetDefault("log_output", "-")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("port_number", 8080)
	viper.SetDefault("server_name", "127.0.0.1")
	viper.SetDefault("static_path", "static")
	viper.SetDefault("repository_path", ".")
	viper.SetDefault("read_timeout", 15)
	viper.SetDefault("write_timeout", 60)

	// Bind environment variables to configuration parameters
	viper.SetEnvPrefix("qubicdb")
	viper.BindEnv("port_number")
	viper.BindEnv("server_name")
	viper.BindEnv("read_timeout")
	viper.BindEnv("write_timeout")

	// Set where to look for the configuration file
	viper.SetConfigName("config")
	viper.SetConfigType("json")

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.qutedb/")
	if runtime.GOOS != "windows" {
		viper.AddConfigPath("/etc/qutedb")
	}

	// Read the configuration
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error while reading config file: %s", err))
	}
}

// createConfiguration uses Viper to initialize a Configuration object.
func createConfiguration() *Configuration {
	configureViper()

	var cookieHashKeyStr = viper.GetString("cookie_hash_key")
	cookieHashKey, err := base64.StdEncoding.DecodeString(cookieHashKeyStr)
	if err != nil {
		panic(fmt.Errorf("Unable to decode cookie hash key \"%s\"",
			cookieHashKeyStr))
	}
	hashLen := len(cookieHashKey)
	if hashLen != 32 && hashLen != 64 {
		panic(fmt.Errorf("Invalid cookie hash key, the length is %d instead of 32/64",
			hashLen))
	}

	var cookieBlockKeyStr = viper.GetString("cookie_block_key")
	cookieBlockKey, err := base64.StdEncoding.DecodeString(cookieHashKeyStr)
	if err != nil {
		panic(fmt.Errorf("Unable to decode cookie block key \"%s\"",
			cookieBlockKeyStr))
	}

	blockLen := len(cookieBlockKey)
	if blockLen != 32 && blockLen != 64 {
		panic(fmt.Errorf("Invalid cookie block key, the length is %d instead of 32/64",
			blockLen))
	}

	return &Configuration{
		ConfigurationFileName: viper.ConfigFileUsed(),
		DatabaseFile:          viper.GetString("database_file"),
		LogFormat:             viper.GetString("log_format"),
		LogLevel:              viper.GetString("log_level"),
		LogOutput:             viper.GetString("log_output"),
		PortNumber:            viper.GetInt("port_number"),
		ReadTimeout:           viper.GetInt64("read_timeout"),
		WriteTimeout:          viper.GetInt64("write_timeout"),
		RepositoryPath:        viper.GetString("repository_path"),
		ServerName:            viper.GetString("server_name"),
		StaticPath:            viper.GetString("static_path"),
		CookieHashKey:         cookieHashKey,
		CookieBlockKey:        cookieBlockKey,
	}
}
