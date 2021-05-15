package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"go-redirector/errors"
	"go-redirector/mapping"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"

	"github.com/gofiber/template/html"
)

var (
	// BuildSha is the commit sha
	BuildSha string
	// BuildVersion is the version of the build
	BuildVersion string
	// BuildDate of the build
	BuildDate string
)

const (
	// DefaultAppName will be used as the app name
	DefaultAppName = "go-redirector"
	// LogLevel is the env var name to use
	LogLevel = "LOG_LEVEL"
	// MappingPath is the env var name to use
	MappingPath = "MAPPING_PATH"
	// Port is the env var name to use
	Port = "PORT"
	// PerformanceMode is the env var name to use
	PerformanceMode = "PERFORMANCE_MODE"
	// HTTPMode is the env var name to use
	HTTPMode = "HTTP_MODE"
	// ServerCert is the env var name to use
	ServerCert = "SERVER_CERT"
	// ServerKey is the env var name to use
	ServerKey = "SERVER_KEY"

	// DefaultLogLevel is the default log level to use
	DefaultLogLevel = zerolog.DebugLevel
	// DefaultMappingPath is the default mapping file to use
	DefaultMappingPath = "./redirect-map.yml"
	// DefaultPort is the default port to use
	DefaultPort = 8080
	// DefaultPortTLS is the default tls port to use
	DefaultPortTLS = 8443
	// DefaultServerCert is the default file name for the server cert
	DefaultServerCert = "./certs/server.pem"
	// DefaultServerKey is the default file name for the server cert key
	DefaultServerKey = "./certs/server.key"
)

// ExitFunc is a function type which can be used for exiting the application
type ExitFunc func(code int)

// Config is a struct representing the configuration of the app
type Config struct {
	LogLevel        zerolog.Level
	MappingPath     string
	Port            int
	MappingsFile    *mapping.MappingsFile
	PerformanceMode bool
	UseHTTP         bool
	ServerCert      string
	ServerKey       string
	exitFunc        ExitFunc
}

func (c *Config) setPerformance(performanceMode bool) {
	c.PerformanceMode = performanceMode
	if performanceMode {
		log.Info().Msg("Performance Mode Enabled, overriding to HTTP mode")
		c.setHTTP(true, "", "")
		c.setLogLevel("error")
	}
}

func (c *Config) setHTTP(useHTTP bool, cert string, key string) {
	c.UseHTTP = useHTTP
	if !useHTTP {
		c.ServerCert = cert
		c.ServerKey = key
		log.Info().Msg("TLS Mode Enabled")
	}
}

func (c *Config) setPort(port int) {
	if port == 0 && c.UseHTTP { // use default tls port
		c.Port = DefaultPort
	} else if port == 0 && !c.UseHTTP {
		c.Port = DefaultPortTLS
	} else {
		c.Port = port // use what user specified
	}
}

func (c *Config) setMappingFile(filePath string) {
	if filePath != "" {
		c.MappingPath = filePath // change it
	}

	// use the mapping file
	if mappingFile, err := mapping.LoadMappingFile(c.MappingPath); err != nil {
		log.Error().Msg(fmt.Sprintf("Bad mapping file: %v", err))
		c.exitFunc(errors.ExitCodeBadMappingFile)
	} else {
		c.MappingsFile = mappingFile
	}
}

func (c *Config) setLogLevel(logLevel string) {
	if level, err := zerolog.ParseLevel(strings.ToLower(logLevel)); err != nil {
		log.Error().Msg(fmt.Sprintf("Error: %v", err))
		c.exitFunc(errors.ExitCodeInvalidLoglevel)
	} else {
		c.LogLevel = level
		zerolog.SetGlobalLevel(level)
	}
}

// SetPort sets the port which should be used by the app
func (c *Config) SetPort(port string) {
	if aPort, err := strconv.Atoi(port); err != nil {
		c.exitFunc(errors.ExitCodeBadPort)
	} else {
		c.Port = aPort
	}
}

func setMappingPath() string {
	logPath := os.Getenv(MappingPath)
	if len(logPath) <= 0 { // if not set give default
		return DefaultMappingPath
	}

	return logPath
}

func goExit(code int) {
	os.Exit(code)
}

// NewConfig generates a new Config
func NewConfig() *Config {
	mappingPath := setMappingPath()

	return &Config{
		MappingPath: mappingPath,
		Port:        DefaultPort,
		exitFunc:    goExit,
	}
}

// LoadEnvPaths loads the env from files starting at HOME, then to local directory.
// Then creates a config object.
func LoadEnvPaths(local string, home string) *Config {
	loadEnv := func(fileName string) bool {
		// load env file first, try home
		if _, err := os.Stat(fileName); err == nil {
			if err := godotenv.Load(fileName); err != nil {
				log.Fatal().Msg(fmt.Sprintf("Error loading .env file %s", fileName))
			}
			return true
		}
		return false
	}

	if !loadEnv(home) { // load from home
		loadEnv(local) // load local, else move on
	}

	return NewConfig()
}

// LoadEnv loads the environment, calling the LoadEnvPaths function with
// preset values for local and home.
func LoadEnv() *Config {
	local := "./.env"
	home := fmt.Sprintf("%s/%s", os.Getenv("HOME"), local)
	return LoadEnvPaths(local, home)
}

// TemplateData is a simple struct to handle redirects
type TemplateData struct {
	RedirectURI string
}

// NewTemplateData returns a struct with all the values needed for templates
func NewTemplateData(redirectURI string) *TemplateData {
	return &TemplateData{RedirectURI: redirectURI}
}

// FastServer represents the server app
type FastServer struct {
	Config      *Config
	MappingFile *mapping.MappingsFile
	server      *fiber.App
	//PrometheusExporter *prometheus.Exporter
}

/**
Respond to health only if host is localhost. Simple guard.
Rely on metrics in future for stats.
Systems deploying (docker, k8) can craft headers with localhost in probes.
*/
func (f *FastServer) healthy(c *fiber.Ctx) error {
	if f.parseHost(c.Hostname()) == "localhost" {
		return c.SendStatus(200)
	}

	return c.SendStatus(404)
}

func (f *FastServer) metrics(c *fiber.Ctx) error {
	if f.parseHost(c.Hostname()) == "localhost" {
		return c.SendStatus(200)
	}

	return c.SendStatus(404)
}

func (f *FastServer) notfound(c *fiber.Ctx) error {
	host := f.parseHost(c.Hostname())
	uri := string(c.Request().URI().Path())
	remoteAddr := c.IP()
	userAgent := c.Get("User-Agent")

	log.Info().Msg(fmt.Sprintf("Returning 404 for requested page [%s%s], by remote client [%s] with user-agent: [%s]",
		host, uri, remoteAddr, userAgent,
	))

	return c.SendStatus(404)
}

func (f *FastServer) index(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	host := f.parseHost(c.Hostname())
	uri := string(c.Request().URI().Path())
	remoteAddr := c.IP()
	userAgent := c.Get("User-Agent")
	scheme := string(c.Request().URI().Scheme())
	mappingEntry, err := f.MappingFile.GetMappingEntry(host, uri)

	// Can't find, return 404
	if err != nil {
		log.Info().Msg(fmt.Sprintf("Request not found for [%s%s], remote client [%s] with user-agent: [%s]",
			host, uri, remoteAddr, userAgent,
		))
		// No content, just hang up with a http code right now.
		return c.SendStatus(404)
	}

	if mappingEntry.Immediate {
		log.Info().Msg(fmt.Sprintf("Redirecting directly to [%s%s] from [%s://%s%s] for remote client [%s] with user-agent: [%s]",
			mappingEntry.Redirect, uri, scheme, c.Hostname(), uri, remoteAddr, userAgent,
		))

		targetURI := fmt.Sprintf("%s%s", mappingEntry.Redirect, uri)
		statusCode := 302
		return c.Redirect(targetURI, statusCode) //nolint
	}

	log.Info().Msg(fmt.Sprintf("Friendly redirect to [%s%s] from [%s://%s%s] for remote client [%s] with user-agent: [%s]",
		mappingEntry.Redirect, uri, scheme, c.Hostname(), uri, remoteAddr, userAgent,
	))
	data := NewTemplateData(mappingEntry.Redirect)
	return c.Render("html", data)
}

func (f *FastServer) parseHost(host string) string {
	if strings.Contains(host, ":") {
		return strings.Split(host, ":")[0]
	}

	return host
}

/**
Bootstrap routes
*/
func (f *FastServer) setup() *fiber.App {
	engine := html.New("./views", ".tpl") // golang template
	server := fiber.New(fiber.Config{
		Views: engine,
		//Prefork: true,  // not right now ...
		ServerHeader: "PlanetVegeta",
		//ProxyHeader: "X-Forwarded-For",
		GETOnly:               true,
		DisableStartupMessage: f.Config.PerformanceMode, // only show banner during perf mode so we can see ps and pid IDs
	})

	server.Use(favicon.New())

	server.Get("/favicon", f.notfound)
	server.Get("/healthy", f.healthy)
	server.Get("/metrics", f.metrics)
	server.Get("/*", f.index)

	f.server = server
	return server
}

// Serve will serve the FastServer on the user defined `port`.
func (f *FastServer) Serve() error {
	server := f.setup()
	port := f.Config.Port

	if f.Config.UseHTTP {
		if err := server.Listen(fmt.Sprintf(":%d", port)); err != nil {
			return err
		}
	} else {
		if err := server.ListenTLS(fmt.Sprintf(":%d", port),
			f.Config.ServerCert,
			f.Config.ServerKey); err != nil {
			return err
		}
	}

	return nil
}

// NewFastServer factory generates a new FastServer
func NewFastServer(config *Config, mappingFile *mapping.MappingsFile) *FastServer {
	return &FastServer{config, mappingFile, fiber.New()}
}

func createServer(c *cli.Context) *FastServer {
	config := LoadEnv() // we load env variable settings first, commandline params may override
	// Must set these first
	config.setLogLevel(c.String("log-level"))
	config.setHTTP(c.Bool("http"), c.String("cert"), c.String("key"))
	config.setPerformance(c.Bool("performance-mode"))

	// config.SetTemplateFromFile(c.String("template"))
	config.setMappingFile(c.String("file"))
	config.setPort(c.Int("port"))

	log.Info().Msg(fmt.Sprintf("Loaded mappings for [%d] host(s).", len(config.MappingsFile.Mappings)))
	log.Info().Msg(fmt.Sprintf("Running server on port [%d].", config.Port))
	//fmt.Printf("%s", config.MappingsFile.Mappings)

	server := NewFastServer(config, config.MappingsFile)

	return server
}

func getAppCommands() []cli.Command {
	var commands = []cli.Command{
		{
			Name:    "run",
			Aliases: []string{"r"},
			Usage:   "run go-redirector",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "log-level, l",
					EnvVar: LogLevel,
					Value:  DefaultLogLevel.String(),
					Usage:  "Log level of the app `LOG_LEVEL`",
				},
				cli.BoolFlag{
					Name:   "http",
					EnvVar: HTTPMode,
					Usage:  "runs in http mode rather than TLS, defaults to port 8080 unless you change it",
				},
				cli.StringFlag{
					Name:   "file, f",
					EnvVar: MappingPath,
					Value:  DefaultMappingPath,
					Usage:  "Use the mapping file specified",
				},
				cli.IntFlag{
					Name:   "port, p",
					EnvVar: Port,
					Usage:  fmt.Sprintf("port to listen on, defaults to %d", DefaultPort),
				},
				cli.BoolFlag{
					Name:   "performance-mode",
					EnvVar: PerformanceMode,
					Usage:  "overrides user supplied flags to allow better performance",
				},
				cli.StringFlag{
					Name:   "cert",
					EnvVar: ServerCert,
					Value:  DefaultServerCert,
					Usage:  "Server Cert to use when TLS mode is enabled",
				},
				cli.StringFlag{
					Name:   "key",
					EnvVar: ServerKey,
					Value:  DefaultServerKey,
					Usage:  "Server Key to use when TLS mode is enabled",
				},
			},
			Action: func(c *cli.Context) error {
				server := createServer(c)
				return server.Serve()
			},
		},
	}

	return commands
}

func newApp(appCommands []cli.Command) *cli.App {
	app := cli.NewApp()
	app.Name = DefaultAppName
	app.Usage = DefaultAppName
	app.Commands = appCommands
	app.Version = fmt.Sprintf("info\n version: %s\n commit: %s\n built: %s",
		BuildVersion, BuildSha, BuildDate)
	return app
}

// Run is the function which should be called by main to start the entire app
func Run(args []string) {
	appCommands := getAppCommands()
	app := newApp(appCommands)

	// Bail if any errors
	err := app.Run(args)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Exiting due to error: %s", err))
	}
}

func main() {
	Run(os.Args)
}
