package configure

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func checkErr(err error) {
	if err != nil {
		logrus.WithError(err).Fatal("config")
	}
}

func New() *Config {
	config := viper.New()

	// Default config
	b, _ := json.Marshal(Config{
		ConfigFile: "config.yaml",
	})
	defaultConfig := bytes.NewReader(b)
	tmp := viper.New()
	tmp.SetConfigType("json")
	checkErr(tmp.ReadConfig(defaultConfig))
	checkErr(config.MergeConfigMap(viper.AllSettings()))

	pflag.String("config", "config.yaml", "Config file location")
	pflag.Bool("noheader", false, "Disable the startup header")
	pflag.Parse()
	checkErr(config.BindPFlags(pflag.CommandLine))

	// File
	config.SetConfigFile(config.GetString("config"))
	config.AddConfigPath(".")
	err := config.ReadInConfig()
	if err == nil {
		checkErr(config.MergeInConfig())
	}

	BindEnvs(config, Config{})

	// Environment
	config.AutomaticEnv()
	config.SetEnvPrefix("REST")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AllowEmptyEnv(true)

	// Print final config
	c := &Config{}
	checkErr(config.Unmarshal(c))

	initLogging(c.Level)

	return c
}

func BindEnvs(config *viper.Viper, iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			BindEnvs(config, v.Interface(), append(parts, tv)...)
		default:
			_ = config.BindEnv(strings.Join(append(parts, tv), "."))
		}
	}
}

type Config struct {
	Level      string `mapstructure:"level" json:"level"`
	ConfigFile string `mapstructure:"config" json:"config"`
	WebsiteURL string `mapstructure:"website_url" json:"website_url"`
	NodeName   string `mapstructure:"node_name" json:"node_name"`
	TempFolder string `mapstructure:"temp_folder" json:"temp_folder"`
	NoHeader   bool   `mapstructure:"noheader" json:"noheader"`

	Redis struct {
		URI      string `mapstructure:"uri" json:"uri"`
		Username string `mapstructure:"username" json:"username"`
		Password string `mapstructure:"password" json:"password"`
		Database int    `mapstructure:"db" json:"db"`
	} `mapstructure:"redis" json:"redis"`

	Mongo struct {
		URI string `mapstructure:"uri" json:"uri"`
		DB  string `mapstructure:"db" json:"db"`
	} `mapstructure:"mongo" json:"mongo"`

	Http struct {
		URI          string `mapstructure:"uri" json:"uri"`
		Type         string `mapstructure:"type" json:"type"`
		CookieDomain string `mapstructure:"cookie_domain" json:"cookie_domain"`
		CookieSecure bool   `mapstructure:"cookie_secure" json:"cookie_secure"`
	} `mapstructure:"http" json:"http"`

	Platforms struct {
		Twitch struct {
			ClientID     string `mapstructure:"client_id" json:"client_id"`
			ClientSecret string `mapstructure:"client_secret" json:"client_secret"`
			RedirectURI  string `mapstructure:"redirect_uri" json:"redirect_uri"`
		} `mapstructure:"twitch" json:"twitch"`
	} `mapstructure:"platforms" json:"platforms"`

	Credentials struct {
		PrivateKey string `mapstructure:"private_key" json:"private_key"`
		PublicKey  string `mapstructure:"public_key" json:"public_key"`
		JWTSecret  string `mapstructure:"jwt_secret" json:"jwt_secret"`
	} `mapstructure:"credentials" json:"credentials"`

	Rmq struct {
		ServerURL       string `mapstructure:"server_url" json:"server_url"`
		JobQueueName    string `mapstructure:"job_queue_name" json:"job_queue_name"`
		ResultQueueName string `mapstructure:"result_queue_name" json:"result_queue_name"`
		UpdateQueueName string `mapstructure:"update_queue_name" json:"update_queue_name"`
	} `mapstructure:"rmq" json:"rmq"`

	Aws struct {
		AccessToken string `mapstructure:"access_token" json:"access_token"`
		SecretKey   string `mapstructure:"secret_key" json:"secret_key"`
		Region      string `mapstructure:"region" json:"region"`
		Bucket      string `mapstructure:"bucket" json:"bucket"`
		Endpoint    string `mapstructure:"endpoint" json:"endpoint"`
	} `mapstructure:"aws" json:"aws"`
}
