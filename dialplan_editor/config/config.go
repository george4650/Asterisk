package config

import (
	"log"

	"github.com/spf13/viper"
)

type (
	Config struct {
		HttpPort int `mapstructure:"HTTP_PORT"`

		SshHost     string `mapstructure:"SSH_HOST"`
		SshPort     int    `mapstructure:"SSH_PORT"`
		SshUser     string `mapstructure:"SSH_USER"`
		SshPassword string `mapstructure:"SSH_PASSWORD"`
	}
)

func LoadConfig() (config Config, err error) {

	viper.AutomaticEnv()

	config.HttpPort = viper.GetInt("HTTP_PORT")

	config.SshHost = viper.GetString("SSH_HOST")
	config.SshPort = viper.GetInt("SSH_PORT")
	config.SshUser = viper.GetString("SSH_USER")
	config.SshPassword = viper.GetString("SSH_PASSWORD")

	return config, err
}

func LoadConfigFile(path string) (config Config, err error) {

	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		log.Fatal("Не удалось загрузить локальный config файл ", err)
	}

	err = viper.Unmarshal(&config)
	return config, nil
}
