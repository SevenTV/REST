package configure

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
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

	// File
	config.SetConfigFile(config.GetString("config_file"))
	config.AddConfigPath(".")
	err := config.ReadInConfig()
	if err != nil {
		logrus.Warning(err)
		logrus.Info("Using default config")
	} else {
		checkErr(config.MergeInConfig())
	}

	// Environment
	replacer := strings.NewReplacer(".", "_")
	config.SetEnvKeyReplacer(replacer)
	config.SetEnvPrefix("7TV")
	config.AllowEmptyEnv(true)
	config.AutomaticEnv()

	// Print final config
	c := &Config{
		viper: config,
	}
	checkErr(config.Unmarshal(c))

	InitLogging(c.Level)

	return c
}

type Config struct {
	Level      string `mapstructure:"level" json:"level"`
	ConfigFile string `mapstructure:"config_file" json:"config_file"`

	WebsiteURL string `mapstructure:"website_url" json:"website_url"`

	NodeName string `mapstructure:"node_name"`

	Redis struct {
		URI string `mapstructure:"uri" json:"uri"`
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

	viper *viper.Viper
}

func (c *Config) Save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	tmp := viper.New()
	tmp.SetConfigType("json")
	if err := tmp.ReadConfig(bytes.NewBuffer(data)); err != nil {
		return err
	}
	if err := c.viper.MergeConfigMap(viper.AllSettings()); err != nil {
		return err
	}

	return c.viper.WriteConfig()
}
