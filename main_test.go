package main

import (
	"github.com/sirupsen/logrus"
	"go-redirector/errors"
	"net/http/httptest"
	"os"
	"testing"
)

func Test_ConfigFactory(t *testing.T) {
	config := NewConfig()
	if config.MappingPath != DEFAULT_MAPPING_PATH {
		t.Errorf("Expected the mapping path to be the default path %s", DEFAULT_MAPPING_PATH)
	}
	if config.Port != DEFAULT_PORT {
		t.Errorf("Expected the default port to %d", DEFAULT_PORT)
	}
}

func Test_ConfigAndPorts(t *testing.T) {
	config := NewConfig()
	config.setPerformance(true)
	config.useHttp(false, "./cert", "./key")

	if !config.PerformanceMode {
		t.Errorf("Expected performance mode not to be %v", config.PerformanceMode)
	}

	if config.UseHttp {
		t.Errorf("Expected to see http mode disabled, instead found it enabled")
	}

	config.setPort(0)
	if config.Port != DEFAULT_PORT_TLS {
		t.Errorf("Expected to see default port of %d, instead found %d", DEFAULT_PORT, config.Port)
	}

	config.setPort(1000)
	if config.Port != 1000 {
		t.Errorf("Expected port to be 1000, instead found %d", config.Port)
	}

	config.useHttp(true, "", "")
	config.setPort(0)
	if config.Port != DEFAULT_PORT {
		t.Errorf("Expected to see default port of %d, instead found %d", DEFAULT_PORT, config.Port)
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
		if code != errors.EXIT_CODE_BAD_MAPPING_FILE {
			t.Errorf("Expected exit code of [%v], got [%v]", errors.EXIT_CODE_BAD_MAPPING_FILE, code)
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
	if config.LogLevel != logrus.DebugLevel {
		t.Errorf("Expected to see logging level set to DEBUG, but saw %v", config.LogLevel)
	}

	// Now test bad values
	exitReached := false
	config.exitFunc = func(code int) {
		if code != errors.EXIT_CODE_INVALID_LOGLEVEL {
			t.Errorf("Expected exit code of [%v], got [%v]", errors.EXIT_CODE_INVALID_LOGLEVEL, code)
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
		if code != errors.EXIT_CODE_BAD_PORT {
			t.Errorf("Expected exit code of [%v], got [%v]", errors.EXIT_CODE_BAD_PORT, code)
		}
	}
	config.SetPort("TRASH")
}

func Test_SetMappingPath(t *testing.T) {
	goodTestFile := "./tests/test-redirect-map.yml"
	//config := NewConfig()
	if path := setMappingPath(); path != DEFAULT_MAPPING_PATH {
		t.Errorf("Expected function to return [%v]", DEFAULT_MAPPING_PATH)
	}

	os.Setenv("MAPPING_PATH", goodTestFile)
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
		os.Unsetenv(testKey)
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
		os.Unsetenv(testKey)
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
