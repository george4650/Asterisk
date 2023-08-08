package main

import (
	"myapp/config"
	"myapp/internal/app"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// @title			Swagger API
// @version		1.0
// @description	Swagger API for Golang Project
func main() {

	// Настройка логгера
	output := zerolog.ConsoleWriter{
		TimeFormat: "02.01.2006 15:04:05",
		Out:        os.Stdout,
	}
	log.Logger = log.Output(output)

	// Configuration
	var conf config.Config
	if _, err := os.Stat("./config/app.env"); err == nil {
		log.Info().Msg("Обнаружен локальный файл конфига. Грузим настройки из него")
		conf, err = config.LoadConfigFile("./config")
		if err != nil {
			log.Err(err)
		}
	} else {
		conf, _ = config.LoadConfig()
	}

	// Run
	app.Run(conf)
}
