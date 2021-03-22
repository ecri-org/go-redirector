package main

import (
	//"context"
	//"contrib.go.opencensus.io/exporter/prometheus"
	//"go.opencensus.io/stats"
	//"go.opencensus.io/stats/view"
	//"go.opencensus.io/tag"

	//"contrib.go.opencensus.io/exporter/prometheus"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"go-redirector/errors"
	"go-redirector/mapping"
	//"go.opencensus.io/stats"
	//"go.opencensus.io/stats/view"
	//"go.opencensus.io/tag"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/template/html"
)

var (
	BUILD_SHA     string
	BUILD_VERSION string
	BUILD_DATE    string

	//MRedirectCounts = stats.Int64("redirect/counts", "The distribution of redirects", "By")
	//KeyHost, _    = tag.NewKey("host")
	//KeyUri, _     = tag.NewKey("uri")
	//RedirectCountView = &view.View{
	//	Name:        "redirect/counts",
	//	Measure:     MRedirectCounts,
	//	Description: "The number of redirects served",
	//	Aggregation: view.Count(),
	//	TagKeys:     []tag.Key{KeyHost, KeyUri},
	//}
)

const (
	LOG_LEVEL    = "LOG_LEVEL"
	MAPPING_PATH = "MAPPING_PATH"
	PORT         = "PORT"
	PERFORMANCE_MODE = "PERFORMANCE_MODE"

	DEFAULT_LOG_LEVEL = log.DebugLevel
	DEFAULT_MAPPING_PATH = "./redirect-map.yml"
	DEFAULT_PORT = 8080
)

type Config struct {
	LogLevel     log.Level
	MappingPath  string
	templateFile string
	Port         int
	MappingsFile *mapping.MappingsFile
	PerformanceMode bool
}

func (c *Config) setPerformance(performanceMode bool) {
	c.PerformanceMode = performanceMode
	if performanceMode {
		log.Info("Performance Mode Enabled")
	}
}

func (c *Config) setPort(port int) {
	c.Port = port
}

func (c *Config) setMappingFile(filePath string) {
	if filePath != "" {
		c.MappingPath = filePath  // change it
	}

	// use the mapping file
	if mappingFile, err := mapping.LoadMappingFile(c.MappingPath); err != nil {
		log.Errorf("Bad mapping file: %v", err)
		os.Exit(errors.EXIT_CODE_BAD_MAPPING_FILE)
	} else {
		c.MappingsFile = mappingFile
	}
}

func (c *Config) setLogLevel(logLevel string) {
	if level, err := log.ParseLevel(logLevel); err != nil {
		log.Errorf("Error: %v", err)
		os.Exit(errors.EXIT_CODE_INVALID_LOGLEVEL)
	} else {
		c.LogLevel = level
		log.SetLevel(level)
		log.SetFormatter(&log.JSONFormatter{})
	}
}

func (c *Config) SetPort(port string) {
	if aPort, err := strconv.Atoi(port); err != nil {
		os.Exit(errors.EXIT_CODE_BAD_PORT)
	} else {
		c.Port = aPort
	}
}

func setMappingPath() string {
	if logPath := os.Getenv(MAPPING_PATH); len(logPath) <= 0 { // if not set give default
		return DEFAULT_MAPPING_PATH
	} else {
		return logPath
	}
}

func NewConfig() *Config {
	mappingPath := setMappingPath()

	return &Config{
		MappingPath: mappingPath,
		Port: DEFAULT_PORT,
	}
}

/**
Loads ENV from file starting from HOME, then to local directory.
Then creates a config object.
 */
func LoadEnv() *Config {
	// env info
	local := fmt.Sprintf("./.env")
	home := fmt.Sprintf("%s/%s", os.Getenv("HOME"), ".env")

	loadEnv := func(fileName string) bool {
		// load env file first, try home
		if _, err := os.Stat(fileName); err == nil {
			if err := godotenv.Load(fileName); err != nil {
				log.Fatalf("Error loading .env file %s", fileName)
			}
			return true
		}
		return false
	}

	if !loadEnv(home) {  // load from home
		loadEnv(local) // load local, else move on
	}

	return NewConfig()
}

type TemplateData struct {
	RedirectUri string
}

func NewTemplateData(redirectUri string) *TemplateData {
	return &TemplateData{RedirectUri: redirectUri}
}

type FastServer struct {
	Config *Config
	MappingFile *mapping.MappingsFile
	//PrometheusExporter *prometheus.Exporter
}

func (f *FastServer) healthy(c *fiber.Ctx) error {
	if f.parseHost(c.Hostname()) == "localhost" {
		return c.SendStatus(200)
	} else {
		return c.SendStatus(404)
	}
}

func (f *FastServer) metrics(c *fiber.Ctx) error {
	if f.parseHost(c.Hostname()) == "localhost" {
		return c.SendStatus(200)
	} else {
		return c.SendStatus(404)
	}
}

func (f *FastServer) notfound(c *fiber.Ctx) error {
	host := f.parseHost(c.Hostname())
	uri := string(c.Request().URI().Path())
	remoteAddr := c.IP()
	userAgent := c.Get("User-Agent")

	log.Infof("Returning 404 for requested page [%s%s], by remote client [%s] with user-agent: [%s]",
		host, uri, remoteAddr, userAgent,
	)

	return c.SendStatus(404)
}

func (f *FastServer) index(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	host := f.parseHost(c.Hostname())
	uri := string(c.Request().URI().Path())
	remoteAddr := c.IP()
	userAgent := c.Get("User-Agent")
	scheme := string(c.Request().URI().Scheme())

	if redirectUri := f.MappingFile.GetRedirectUri(host, uri); redirectUri == "" {
		log.Infof("Request not found for [%s%s], remote client [%s] with user-agent: [%s]",
			host, uri, remoteAddr, userAgent,
		)
		// No content, just hang up with a http code right now.
		return c.SendStatus(404)
	} else {
		//ctx, _ := tag.New(context.Background(), tag.Insert(KeyHost, host), tag.Insert(KeyUri, uri))

		//defer func() {
		//	stats.Record(ctx, MRedirectCounts.M(1))
		//}()
		log.Infof("Redirecting to [%s%s] from [%s://%s%s] for remote client [%s] with user-agent: [%s]",
			redirectUri, uri, scheme, c.Hostname(), uri, remoteAddr, userAgent,
		)

		data := NewTemplateData(redirectUri)
		return c.Render("html", data)
	}
}

func (f *FastServer) parseHost(host string) string {
	if strings.Contains(host, ":") {
		return strings.Split(host, ":")[0]
	} else {
		return host
	}
}

func (f *FastServer) Serve(port int) error {
	engine := html.New("./views", ".tpl") // golang template
	server := fiber.New(fiber.Config{
		Views: engine,
		//Prefork: true,  // not right now ...
		ServerHeader: "PlanetVegeta",
		//ProxyHeader: "X-Forwarded-For",
		GETOnly: true,
	})

	server.Use(favicon.New())

	server.Get("/favicon", f.notfound)
	server.Get("/healthy", f.healthy)
	server.Get("/metrics", f.metrics)
	server.Get("/*", f.index)

	if err := server.Listen(fmt.Sprintf(":%d", port)); err != nil {
		return err
	}

	return nil
}

func NewFastServer(config *Config, mappingFile *mapping.MappingsFile) *FastServer {
	//exporter, err := prometheus.NewExporter(prometheus.Options{
	//	Namespace: "go-redirector",
	//})

	//if err != nil {
	//	log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	//	os.Exit(errors.EXIT_METRICS_ISSUE)
	//}

	//return &FastServer{config, mappingFile, exporter}
	return &FastServer{config, mappingFile}
}

func Run(args []string) {
	var AppCommands = []cli.Command{
		{
			Name:    "run",
			Aliases: []string{"r"},
			Usage:   "run go-redirector",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "log-level, l",
					EnvVar: LOG_LEVEL,
					Value: DEFAULT_LOG_LEVEL.String(),
					Usage: "Log level of the app `LOG_LEVEL`",
				},
				cli.StringFlag{
					Name:  "file, f",
					EnvVar: MAPPING_PATH,
					Value: DEFAULT_MAPPING_PATH,
					Usage: "Use the mapping file specified",
				},
				cli.IntFlag{
					Name:  "port, p",
					EnvVar: PORT,
					Value: DEFAULT_PORT,
					Usage: fmt.Sprintf("port to listen on, defaults to %d", DEFAULT_PORT),
				},
				cli.BoolFlag{
					Name:  "performance-mode",
					EnvVar: PERFORMANCE_MODE,
					Usage: "run using a faster templating system",
				},
			},
			Action: func(c *cli.Context) error {
				config := LoadEnv()  // we load env variable settings first, commandline params may override
				// Must set these first
				config.setLogLevel(c.String("log-level"))
				config.setPerformance(c.Bool("performance-mode"))

				// config.SetTemplateFromFile(c.String("template"))
				config.setMappingFile(c.String("file"))
				config.setPort(c.Int("port"))

				log.Infof("Loaded [%d] redirect mappings.", len(config.MappingsFile.Mappings))
				log.Infof("Running server on port [%d].", config.Port)

				//if err := view.Register(RedirectCountView); err != nil {
				//	log.Fatalf("Failed to register views: %v", err)
				//}

				server := NewFastServer(config, config.MappingsFile)
				return server.Serve(config.Port)
			},
		},
	}

	app := cli.NewApp()
	app.Name = "go-redirector"
	app.Usage = "go-redirector"
	app.Commands = AppCommands
	app.Version = fmt.Sprintf("info\n version: %s\n commit: %s\n built: %s",
		BUILD_VERSION, BUILD_SHA, BUILD_DATE)

	// Bail if any errors
	err := app.Run(args)
	if err != nil {
		log.Fatalf("Exiting due to error: %s", err)
	}
}

func main() {
	Run(os.Args)
}