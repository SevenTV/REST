package configure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/kr/pretty"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Cfg struct {
	Level      string `mapstructure:"level" json:"level"`
	ConfigFile string `mapstructure:"config_file" json:"config_file"`
}

// default config
var defaultConf = Cfg{
	ConfigFile: "config.yaml",
}

var Config = viper.New()

// Capture environment variables
var NodeName string = os.Getenv("NODE_NAME")
var PodName string = os.Getenv("POD_NAME")
var PodIP string = os.Getenv("POD_IP")

func initLog() {
	if l, err := log.ParseLevel(Config.GetString("level")); err == nil {
		log.SetLevel(l)
		log.SetReportCaller(true)
	}
}

func checkErr(err error) {
	if err != nil {
		log.WithError(err).Fatal("config")
	}
}

func init() {
	if NodeName == "" {
		NodeName = "STANDALONE"
	}
	if PodName == "" {
		PodName = "STANDALONE"
	}

	log.SetFormatter(&log.JSONFormatter{})
	// Default config
	b, _ := json.Marshal(defaultConf)
	defaultConfig := bytes.NewReader(b)
	viper.SetConfigType("json")
	checkErr(viper.ReadConfig(defaultConfig))
	checkErr(Config.MergeConfigMap(viper.AllSettings()))

	// File
	Config.SetConfigFile(Config.GetString("config_file"))
	Config.AddConfigPath(".")
	err := Config.ReadInConfig()
	if err != nil {
		log.Warning(err)
		log.Info("Using default config")
	} else {
		checkErr(Config.MergeInConfig())
	}

	// Environment
	replacer := strings.NewReplacer(".", "_")
	Config.SetEnvKeyReplacer(replacer)
	Config.AllowEmptyEnv(true)
	Config.AutomaticEnv()

	// Log
	initLog()

	// Print final config
	c := Cfg{}
	checkErr(Config.Unmarshal(&c))
	log.Debugf("Current configurations: \n%# v", pretty.Formatter(c))

	Config.WatchConfig()
	Config.OnConfigChange(func(_ fsnotify.Event) {
		fmt.Println("Config has changed")
	})
}
