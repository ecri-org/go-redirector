package main

import (
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/urfave/cli"
	"go-redirector/errors"
	"net/http/httptest"
	"os"
	"testing"
)

func Test_ConfigFactory(t *testing.T) {
	config := NewConfig()
	if config.MappingPath != DefaultMappingPath {
		t.Errorf("Expected the mapping path to be the default path %s", DefaultMappingPath)
	}
	if config.Port != DefaultPort {
		t.Errorf("Expected the default port to %d", DefaultPort)
	}
}

func Test_ConfigAndPorts(t *testing.T) {
	config := NewConfig()
	config.setPerformance(true)
	config.setHTTP(false, "./cert", "./key")

	if !config.PerformanceMode {
		t.Errorf("Expected performance mode not to be %v", config.PerformanceMode)
	}

	if config.UseHTTP {
		t.Errorf("Expected to see http mode disabled, instead found it enabled")
	}

	config.setPort(0)
	if config.Port != DefaultPortTLS {
		t.Errorf("Expected to see default port of %d, instead found %d", DefaultPort, config.Port)
	}

	config.setPort(1000)
	if config.Port != 1000 {
		t.Errorf("Expected port to be 1000, instead found %d", config.Port)
	}

	config.setHTTP(true, "", "")
	config.setPort(0)
	if config.Port != DefaultPort {
		t.Errorf("Expected to see default port of %d, instead found %d", DefaultPort, config.Port)
	}

	config.SetPort("1001")
	if config.Port != 1001 {
		t.Errorf("Expected port to be 1001, instead found %d", config.Port)
	}
}

func Test_ExitConfigMapping(t *testing.T) {
	badFile := "../tests/bad-redirect-map.yml"

	// we'll use this to see if the exit was reached
	exitReached := false

	config := NewConfig()
	config.exitFunc = func(code int) {
		if code != errors.ExitCodeBadMappingFile {
			t.Errorf("Expected exit code of [%v], got [%v]", errors.ExitCodeBadMappingFile, code)
		}
		exitReached = true
	}
	config.setMappingFile(badFile)

	if !exitReached {
		t.Errorf("Expected to see the app exit on attempting to load a bad config, it did not.")
	}
}

func Test_ConfigMapping(t *testing.T) {
	goodFile := "./tests/test-redirect-map.yml"
	config := NewConfig()
	config.exitFunc = func(code int) {
		t.Errorf("Did not Expected to see an exception and have the app exit on attempting to load a good config.")
	}
	config.setMappingFile(goodFile)
}

func Test_ConfigLogLevel(t *testing.T) {
	debug := "DEBUG"
	config := NewConfig()

	// First lets test expected values
	config.exitFunc = func(code int) {
		t.Errorf("Did not Expected to see an exception, please see logs and test.")
	}
	config.setLogLevel(debug)
	if config.LogLevel != zerolog.DebugLevel {
		t.Errorf("Expected to see logging level set to DEBUG, but saw %v", config.LogLevel)
	}

	// Now test bad values
	exitReached := false
	config.exitFunc = func(code int) {
		if code != errors.ExitCodeInvalidLoglevel {
			t.Errorf("Expected exit code of [%v], got [%v]", errors.ExitCodeInvalidLoglevel, code)
		}
		exitReached = true
	}
	config.setLogLevel("TRASH")
	if !exitReached {
		t.Errorf("Expected to have caught an exception, but did not.")
	}
}

func Test_SetPort(t *testing.T) {
	config := NewConfig()

	config.exitFunc = func(code int) {
		t.Errorf("Did not Expected to see an exception and have the app exit on attempting to load a good config.")
	}

	config.SetPort("80")
	config.SetPort("8080")
	config.SetPort("443")
	config.SetPort("8443")

	config.exitFunc = func(code int) {
		if code != errors.ExitCodeBadPort {
			t.Errorf("Expected exit code of [%v], got [%v]", errors.ExitCodeBadPort, code)
		}
	}
	config.SetPort("TRASH")
}

func Test_SetMappingPath(t *testing.T) {
	goodTestFile := "./tests/test-redirect-map.yml"
	//config := NewConfig()
	if path := setMappingPath(); path != DefaultMappingPath {
		t.Errorf("Expected function to return [%v]", DefaultMappingPath)
	}

	if err := os.Setenv("MAPPING_PATH", goodTestFile); err != nil {
		t.Errorf("Test harness could not set env var MAPPING_PATH=%s", goodTestFile)
	}

	if path := setMappingPath(); path != goodTestFile {
		t.Errorf("Expected function to return [%v]", goodTestFile)
	}
}

func Test_LoadEnvPaths(t *testing.T) {
	testKey := "URI_REDIRECTOR_TEST"
	local := "tests/.env"
	home := "tests/home/.env"
	badPath := "tests/noop/"
	//badFile := "tests/bad/.env"

	//debugEnv := func() {
	//	for _, e := range os.Environ() {
	//		pair := strings.SplitN(e, "=", 2)
	//		fmt.Println(pair)
	//	}
	//}

	// load home
	config := LoadEnvPaths(local, home)
	if config == nil {
		t.Errorf("Expected config to not be nil")
	} else {
		expected := "home"
		if value := os.Getenv(testKey); value == "" {
			t.Errorf("Expected to see env var read")
		} else if value != expected {
			t.Errorf("Expected env [%s] var read as [%s], instead it was [%s]", testKey, expected, value)
		}
		if err := os.Unsetenv(testKey); err != nil {
			t.Errorf("Test harness could not unset env var %s", testKey)
		}
	}

	// load local
	config = LoadEnvPaths(local, badPath)
	if config == nil {
		t.Errorf("Expected config to not be nil")
	} else {
		expected := "local"
		if value := os.Getenv(testKey); value == "" {
			t.Errorf("Expected to see env var read")
		} else if value != expected {
			t.Errorf("Expected env [%s] var read as [%s], instead it was [%s]", testKey, expected, value)
		}

		if err := os.Unsetenv(testKey); err != nil {
			t.Errorf("Test harness could not unset env var %s", testKey)
		}
	}

	//config = LoadEnvPaths(badFile, badFile)
	//if config != nil {
	//	t.Errorf("Expected to see error trying to load bad file [%s]", badFile)
	//}
}

func Test_LoadEnv(t *testing.T) {
	config := LoadEnv()
	if config == nil {
		t.Errorf("Expected a new instance of config even if no .env files were found.")
	}
}

func Test_NewTemplateData(t *testing.T) {
	if td := NewTemplateData("test"); td == nil {
		t.Errorf("Expected TemplateData, got nil")
	}
}

func Test_FastServerRoutes(t *testing.T) {
	config := NewConfig()
	fastServer := NewFastServer(config, config.MappingsFile)
	fastServer.setup()

	testServer := func(target string) {
		request := httptest.NewRequest("GET", target, nil)

		// Test /healthy with host=localhost, should get 200
		request.Host = "localhost"
		expectedStatusCode := 200
		if resp, err := fastServer.server.Test(request); err != nil {
			t.Errorf("Did not expect to get an error testing target [%s], error: %v", target, err)
		} else {
			if resp.StatusCode != expectedStatusCode {
				t.Errorf("expected [%d], got [%d]", expectedStatusCode, resp.StatusCode)
			}
		}

		// Test /healthy with host=example, should get 404
		request.Host = "example.com"
		expectedStatusCode = 404
		if resp, err := fastServer.server.Test(request); err != nil {
			t.Errorf("Did not expect to get an error testing target [%s], error: %v", target, err)
		} else {
			if resp.StatusCode != expectedStatusCode {
				t.Errorf("expected [%d], got [%d]", expectedStatusCode, resp.StatusCode)
			}
		}
	}

	testServer("/healthy")
	testServer("/metrics")
}

/**
Test routes not found and specifically favicon which we return as 404.
*/
func Test_FastServerNotFoundRoutes(t *testing.T) {
	testFile := "./tests/test-redirect-map.yml"

	config := NewConfig()
	config.setMappingFile(testFile)
	fastServer := NewFastServer(config, config.MappingsFile)
	fastServer.setup()

	target := "/notfound"
	expectedStatusCode := 404
	request := httptest.NewRequest("GET", target, nil)
	request.Host = "localhost"

	if resp, err := fastServer.server.Test(request); err != nil {
		t.Errorf("Did not expect to get an error testing target [%s], error: %v", target, err)
	} else {
		if resp.StatusCode != expectedStatusCode {
			t.Errorf("expected [%d], got [%d]", expectedStatusCode, resp.StatusCode)
		}
	}

	target = "/favicon"
	request = httptest.NewRequest("GET", target, nil)
	request.Host = "testhost"

	if resp, err := fastServer.server.Test(request); err != nil {
		t.Errorf("Did not expect to get an error testing target [%s], error: %v", target, err)
	} else {
		if resp.StatusCode != expectedStatusCode {
			t.Errorf("expected [%d], got [%d]", expectedStatusCode, resp.StatusCode)
		}
	}
}

/**
These routes are set in the test mapping file.
*/
func Test_FastServerMappedRoute(t *testing.T) {
	testFile := "./tests/test-redirect-map.yml"

	config := NewConfig()
	config.setMappingFile(testFile)
	fastServer := NewFastServer(config, config.MappingsFile)
	fastServer.setup()

	target := "/my-path"
	expectedStatusCode := 200

	request := httptest.NewRequest("GET", target, nil)
	request.Host = "testhost"

	if resp, err := fastServer.server.Test(request); err != nil {
		t.Errorf("Did not expect to get an error testing target [%s], error: %v", target, err)
	} else {
		if resp.StatusCode != expectedStatusCode {
			t.Errorf("expected [%d], got [%d]", expectedStatusCode, resp.StatusCode)
		}
	}
}

/**
These routes are set in the test mapping file.
*/
func Test_FastServerRedirectMappedRoute(t *testing.T) {
	testFile := "./tests/test-redirect-map.yml"

	config := NewConfig()
	config.setMappingFile(testFile)
	fastServer := NewFastServer(config, config.MappingsFile)
	fastServer.setup()

	// test friendly: true
	target := "/my-path"
	expectedStatusCode := 200
	request := httptest.NewRequest("GET", target, nil)
	request.Host = "testhost"

	if resp, err := fastServer.server.Test(request); err != nil {
		t.Errorf("Did not expect to get an error testing target [%s], error: %v", target, err)
	} else {
		if resp.StatusCode != expectedStatusCode {
			t.Errorf("expected [%d], got [%d]", expectedStatusCode, resp.StatusCode)
		}
	}

	// test immediate: false
	target = "/direct"
	expectedStatusCode = 302
	request = httptest.NewRequest("GET", target, nil)
	request.Host = "testhost"

	if resp, err := fastServer.server.Test(request); err != nil {
		t.Errorf("Did not expect to get an error testing target [%s], error: %v", target, err)
	} else {
		if resp.StatusCode != expectedStatusCode {
			t.Errorf("expected [%d], got [%d]", expectedStatusCode, resp.StatusCode)
		}
	}
}

func Test_CreateServer(t *testing.T) {
	// Bare minimum required
	fl := cli.StringFlag{
		Name:  "log-level, l",
		Value: DefaultLogLevel.String(),
		Usage: "Log level of the app `LOG_LEVEL`",
	}
	flagSet := flag.NewFlagSet("test", 0)
	fl.Apply(flagSet)

	app := cli.NewApp()
	context := cli.NewContext(app, flagSet, nil)
	if context == nil {
		t.Errorf("bad")
	}

	server := createServer(context)
	if server == nil {
		t.Errorf("bad")
	}
}

func Test_GetAppCommands(t *testing.T) {
	commands := getAppCommands()
	flags := commands[0].Flags

	// carefully match these from the flags in `main.go`
	expectedFlags := []string{
		"log-level, l",
		"http",
		"file, f",
		"port, p",
		"performance-mode",
		"cert",
		"key",
	}

	if len(flags) != len(expectedFlags) {
		t.Errorf("getAppCommands generates %d flags, expectedFlags should match it, len was %d", len(flags), len(expectedFlags))
	}

	found := 0
	for _, flag := range flags {
		for _, e := range expectedFlags {
			flagName := flag.GetName()
			if e == flagName {
				found = found + 1
			}
		}
	}

	if found != len(expectedFlags) {
		t.Errorf("getAppCommands generates %d flags, instead based on expectedFlags only found %d", found, len(expectedFlags))
	}
}

func Test_NewApp(t *testing.T) {
	commands := getAppCommands()
	app := newApp(commands)

	if app.Name != DefaultAppName {
		t.Errorf("Expected to see the default app name as %s, but found %s", DefaultAppName, app.Name)
	}

	if app.Usage != DefaultAppName {
		t.Errorf("Expected to see the default app usage as %s, but found %s", DefaultAppName, app.Name)
	}
}

func Test_ParseHost(t *testing.T) {
	// Bare minimum required
	fl := cli.StringFlag{
		Name:  "log-level, l",
		Value: DefaultLogLevel.String(),
		Usage: "Log level of the app `LOG_LEVEL`",
	}
	flagSet := flag.NewFlagSet("test", 0)
	fl.Apply(flagSet)

	app := cli.NewApp()
	context := cli.NewContext(app, flagSet, nil)
	if context == nil {
		t.Errorf("bad")
	}

	server := createServer(context)

	type hostEntry struct {
		host string
		port string
	}
	type hostList []hostEntry

	cat := func(entry hostEntry) string {
		return fmt.Sprintf("%s:%s", entry.host, entry.port)
	}

	commonHost := "localhost"
	testData := hostList{
		hostEntry{commonHost, "80"},
		hostEntry{commonHost, "8080"},
		hostEntry{commonHost, "8443"},
		hostEntry{commonHost, "443"},
	}

	for _, testEntry := range testData {
		if parsed := server.parseHost(cat(testEntry)); parsed != testEntry.host {
			t.Errorf("Expected host to be [%s]", testEntry.host)
		}
	}

	if parsed := server.parseHost(commonHost); parsed != commonHost {
		t.Errorf("Expected host to be [%s]", commonHost)
	}
}
