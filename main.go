package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gitlab.citicom.kz/CloudServer/server"
	"gitlab.citicom.kz/CloudServer/server/database"
	"gitlab.citicom.kz/CloudServer/server/influx"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	var logDestination string
	flag.StringVar(&logDestination, "ld", "file", "lod destination")
	flag.Parse()

	log.SetFormatter(&log.JSONFormatter{})

	if _, err := os.Stat("." + string(filepath.Separator) + "logs"); err != nil {
		_ = os.Mkdir("."+string(filepath.Separator)+"logs", 0777)
	}
	logFileName := "." + string(filepath.Separator) + "logs" + string(filepath.Separator) + "main.log"
	if logDestination == "file" {
		log.SetOutput(&lumberjack.Logger{
			Filename: logFileName,
			MaxSize:  50, // megabytes
			MaxAge:   10, //days
		})
	}
	log.SetLevel(log.InfoLevel)

	viper.SetDefault("Host", "localhost")
	viper.SetDefault("Port", "9999")
	viper.SetDefault("DatabaseName", "cloud_server")
	viper.SetDefault("DatabaseUser", "root")
	viper.SetDefault("DatabasePass", "To!change!@")
	viper.SetDefault("DatabaseHost", "tcp(localhost:3306)")
	viper.SetDefault("DatabaseCharset", "utf8")

	viper.SetDefault("InfluxDatabaseHost", "http://localhost:8086")
	viper.SetDefault("InfluxDatabaseName", "cloudDB")

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("json")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Infof("Config file changed: %s", e.Name)
	})
}

func main() {
	fmt.Println("TRY START")
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic in main-push function: %v\n", err)
			os.Exit(1)
		}
	}()

	host := viper.GetString("Host")
	port := viper.GetString("Port")
	dbName := viper.GetString("DatabaseName")
	dbLogin := viper.GetString("DatabaseUser")
	dbPass := viper.GetString("DatabasePass")
	dbHost := viper.GetString("DatabaseHost")
	dbCharset := viper.GetString("DatabaseCharset")
	dbConnectString := fmt.Sprintf("%s:%s@%s/%s?charset=%s", dbLogin, dbPass, dbHost, dbName, dbCharset)

	db, err := database.Open(dbConnectString)
	if err != nil {
		fmt.Println("ERROR: ", err)
		log.Errorf("Can't open database: %s", err.Error())
		return
	}

	influxHost := viper.GetString("InfluxDatabaseHost")
	influxDBName := viper.GetString("InfluxDatabaseName")
	influxDB, err := influx.Open(influxHost, influxDBName)
	if err != nil {
		fmt.Println("ERROR INFLUX: ", err)
		log.Println("Can't open database INFLUX: %s", err.Error())
		return
	}

	client := server.NewServer(host, port, db, influxDB)
	client.Run()
}
