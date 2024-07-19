package main

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"log"
	"os"
)

// Config stores the application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

// ServerConfig stores server specific configurations
type ServerConfig struct {
	Port string
	IP   string
}

// DatabaseConfig stores database specific configurations
type DatabaseConfig struct {
	File string
}

var cfg Config
var db *sql.DB
var rootCmd = &cobra.Command{Use: "simpleshorturl"}

func main() {
	cobra.OnInitialize(initConfig, initDB)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	configFile := "config.toml"
	if file := os.Getenv("CONFIG_FILE"); file != "" {
		configFile = file
	}
	if _, err := toml.DecodeFile(configFile, &cfg); err != nil {
		log.Fatalf("Error decoding config file: %s", err)
	}
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", cfg.Database.File)
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		short_mark TEXT NOT NULL,
		original_url TEXT NOT NULL,
		access_count INTEGER DEFAULT 0
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %s", err)
	}
}

func startServer() {
	r := gin.Default()
	r.GET("/:shortMark", func(c *gin.Context) {
		shortMark := c.Param("shortMark")
		var originalURL string
		var id, accessCount int
		err := db.QueryRow("SELECT id, original_url, access_count FROM urls WHERE short_mark = ?", shortMark).Scan(&id, &originalURL, &accessCount)
		if err != nil {
			c.AbortWithStatus(404)
			return
		}
		_, err = db.Exec("UPDATE urls SET access_count = access_count + 1 WHERE id = ?", id)
		if err != nil {
			c.AbortWithStatus(500)
			return
		}
		c.Redirect(301, originalURL)
	})
	r.Run(cfg.Server.IP + cfg.Server.Port) // Listen and serve on defined IP and port from config
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}
