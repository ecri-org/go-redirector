package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"go-redirector/errors"
	"go-redirector/mapping"
	tplhtml "html/template" // this reduces performance via reflection
	"net/http"
	"os"
	"strconv"
	"strings"
	tpltxt "text/template"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"contrib.go.opencensus.io/exporter/prometheus"
)

var (
	BUILD_SHA     string
	BUILD_VERSION string
	BUILD_DATE    string

	MRedirectCounts = stats.Int64("redirect/counts", "The distribution of redirects", "By")
	KeyHost, _    = tag.NewKey("host")
	KeyUri, _     = tag.NewKey("uri")
	RedirectCountView = &view.View{
		Name:        "redirect/counts",
		Measure:     MRedirectCounts,
		Description: "The number of redirects served",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyHost, KeyUri},
	}
)

// When using the templateHtml below, the hidden paragraph at the bottom is unique.
const DEFAULT_HTML_TEMPLATE = `
<html>
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
</head>
<body>
<p>The page you reached has moved to <a href="{{.RedirectUri}}">{{.RedirectUri}}</a>, please update your bookmarks.</p>
<p>You will be automatically redirected to {{.RedirectUri}} in <span id="countdown">15</span> seconds.</p>
<p>Or click <a href="{{.RedirectUri}}">THIS LINK</a> to go there now.</p>
<script type="text/javascript">
	let seconds = 15;

	function countdown() {
		seconds = seconds - 1;
		if (seconds < 0) {
			window.location = "{{.RedirectUri}}";
		} else {
			document.getElementById("countdown").innerHTML = seconds.toString();
			window.setTimeout("countdown()", 1000);
		}
	}
	countdown();
</script>
<p hidden>Generated by simple-redirector.</p>
</body>
</html>
`

const (
	LOG_LEVEL    = "LOG_LEVEL"
	MAPPING_PATH = "MAPPING_PATH"
	PORT         = "PORT"
	PERFORMANCE_MODE = "PERFORMANCE_MODE"

	DEFAULT_LOG_LEVEL = log.DebugLevel
	DEFAULT_MAPPING_PATH = "./redirect-map.yml"
	DEFAULT_TEMPLATE_PATH = "./html.tpl"
	DEFAULT_PORT = 8080
)

type Config struct {
	LogLevel     log.Level
	MappingPath  string
	templateFile string
	templateHtml *tplhtml.Template
	templateText *tpltxt.Template
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

// Proxy
func (c *Config) SetTemplateFromFile(templateFile string) {
	if c.PerformanceMode {
		c.SetTextTemplateFromFile(templateFile)
	} else {
		c.SetHtmlTemplateFromFile(templateFile)
	}
}

// TODO: should use interface instead!
func (c *Config) SetTextTemplateFromFile(templateFile string) {
	useBuiltIn := func() {
		template := tpltxt.New("default")
		if tpl, err := template.Parse(DEFAULT_HTML_TEMPLATE); err != nil {
			log.Fatalf("Could not read templateHtml file [%s]", templateFile)
			os.Exit(errors.EXIT_CODE_TPL_ERROR)
		} else {
			c.templateText = tpl
		}
	}

	useFile := func() {
		c.templateFile = templateFile
		if tpl, err := tpltxt.ParseFiles(templateFile); err != nil {
			log.Fatalf("Could not read templateHtml file [%s]", templateFile)
			os.Exit(errors.EXIT_CODE_TPL_NOT_FOUND)
		} else {
			c.templateText = tpl
		}
	}

	if templateFile == "" {
		useBuiltIn()
	} else {
		useFile()
	}
}

func (c *Config) SetHtmlTemplateFromFile(templateFile string) {
	useBuiltIn := func() {
		template := tplhtml.New("default")
		if tpl, err := template.Parse(DEFAULT_HTML_TEMPLATE); err != nil {
			log.Fatalf("Could not read templateHtml file [%s]", templateFile)
			os.Exit(errors.EXIT_CODE_TPL_ERROR)
		} else {
			c.templateHtml = tpl
		}
	}

	useFile := func() {
		c.templateFile = templateFile
		if tpl, err := tplhtml.ParseFiles(templateFile); err != nil {
			log.Fatalf("Could not read templateHtml file [%s]", templateFile)
			os.Exit(errors.EXIT_CODE_TPL_NOT_FOUND)
		} else {
			c.templateHtml = tpl
		}
	}

	if templateFile == "" {
		useBuiltIn()
	} else {
		useFile()
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
	PrometheusExporter *prometheus.Exporter
}

func (f *FastServer) RenderTemplate(data *TemplateData) (string, error) {
	var tpl bytes.Buffer

	if f.Config.PerformanceMode {
		if err := f.Config.templateText.Execute(&tpl, data); err != nil {
			log.Errorf("Encountered issues rendering templateHtml, %v", err)
			return "", err
		} else {
			return tpl.String(), nil
		}
	} else {
		if err := f.Config.templateHtml.Execute(&tpl, data); err != nil {
			log.Errorf("Encountered issues rendering templateHtml, %v", err)
			return "", err
		} else {
			return tpl.String(), nil
		}
	}
}

func (f *FastServer) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	host := f.parseHost(r.Host)
	log.Infof("Returning 404 for requested page [%s%s], by remote client [%s] with user-agent: [%s]",
		host, r.RequestURI, r.RemoteAddr, r.Header.Get("User-Agent"),
	)
	w.WriteHeader(http.StatusNotFound)
}

func (f *FastServer) mappingHandler(w http.ResponseWriter, r *http.Request) {
	host := f.parseHost(r.Host)
	uri := r.RequestURI

	if redirectUri := f.MappingFile.GetRedirectUri(host, uri); redirectUri == "" {
		log.Infof("Request not found for [%s%s], remote client [%s] with user-agent: [%s]",
			host, uri, r.RemoteAddr, r.Header.Get("User-Agent"),
		)
		// No content, just hang up with a http code right now.
		w.WriteHeader(http.StatusNotFound)
	} else {
		ctx, _ := tag.New(context.Background(), tag.Insert(KeyHost, host), tag.Insert(KeyUri, uri))

		defer func() {
			stats.Record(ctx, MRedirectCounts.M(1))
		}()

		log.Infof("Redirecting to [%s%s] from [%s] for remote client [%s] with user-agent: [%s]",
			redirectUri, uri, host, r.RemoteAddr, r.Header.Get("User-Agent"),
		)

		data := NewTemplateData(redirectUri)
		content, renderError := f.RenderTemplate(data)
		if renderError != nil {
			log.Error(renderError)
			fmt.Fprint(w, renderError)
		} else {
			if _, err := fmt.Fprint(w, content); err != nil {
				log.Error(err)
			}
		}
	}
}

func (f *FastServer) healthy(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (f *FastServer) metrics(w http.ResponseWriter, r *http.Request) {

}

func (f *FastServer) index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Add("Content-Type", "text/html")  // force

	processLocalHost := func() {
		if r.RequestURI == "/healthy" {
			f.healthy(w, r)
		}
		//if r.RequestURI == "/metrics" {
		//	f.PrometheusExporter
		//}
	}

	processRequest := func() {
		if r.RequestURI == "/favicon.ico" {
			f.notFoundHandler(w, r)
		} else {
			f.mappingHandler(w, r)
		}
	}

	if f.parseHost(r.Host) == "localhost" {
		processLocalHost()
	} else {
		processRequest()
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
	router := httprouter.New()
	router.GET("/*path", f.index)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), router); err != nil {
		return err
	}

	return nil
}

func NewFastServer(config *Config, mappingFile *mapping.MappingsFile) *FastServer {
	exporter, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "simple-redirector",
	})

	if err != nil {
		log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
		os.Exit(errors.EXIT_METRICS_ISSUE)
	}

	return &FastServer{config, mappingFile, exporter}
}

func Run(args []string) {
	var AppCommands = []cli.Command{
		{
			Name:    "run",
			Aliases: []string{"r"},
			Usage:   "run simple redirector",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "log-level, l",
					EnvVar: LOG_LEVEL,
					Usage: "Log level of the app `LOG_LEVEL`",
				},
				cli.StringFlag{
					Name:  "file, f",
					EnvVar: MAPPING_PATH,
					Usage: "Use the mapping file specified",
				},
				cli.StringFlag{
					Name:  "template, t",
					EnvVar: DEFAULT_TEMPLATE_PATH,
					Usage: "Use the specified golang templateHtml file, otherwise rely on app provided html",
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

				config.SetTemplateFromFile(c.String("template"))
				config.setMappingFile(c.String("file"))
				config.setPort(c.Int("port"))

				log.Infof("Loaded [%d] redirect mappings.", len(config.MappingsFile.Mappings))
				log.Infof("Running server on port [%d].", config.Port)

				if err := view.Register(RedirectCountView); err != nil {
					log.Fatalf("Failed to register views: %v", err)
				}

				server := NewFastServer(config, config.MappingsFile)
				return server.Serve(config.Port)
			},
		},
	}

	app := cli.NewApp()
	app.Name = "simple-redirector"
	app.Usage = "simple-redirector"
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
