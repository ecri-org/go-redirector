package mapping

import (
	"fmt"
	"testing"
)

/**
These patterns will not pass validation.
 */
var badMappings = []struct{
	host string
	path string
	redirect string
}{
	{
		"localhost",
		"", // empty path
		"https://127.0.0.1",
	},
	{
		"localhost",
		"pathA",
		"://127.0.0.1",  // no scheme
	},
	{
		"localhost",
		"pathA",  // path has no slash prefix
		"https://127.0.0.1",
	},
	{
		"localhost",
		"/pathA",
		"http://127.0.0.1",  // we only accept https, sorry
	},
	{
		"localhost",
		"/pathA",
		"ftp://127.0.0.1",  // we only accept https, sorry
	},
}

func Test_MappingFactory(t *testing.T) {
	path := "/"
	redirect := "127.0.0.1"
	mapping := NewMapping(path, redirect)
	//if mapping.Host != host {
	//	t.Errorf("MappingsFile factory failed, expected host: [%s], got [%s]", host, mapping.Host)
	//}
	if mapping.Path != path {
		t.Errorf("MappingsFile factory failed, expected path: [%s], got [%s]", path, mapping.Path)
	}
	if mapping.Redirect != redirect {
		t.Errorf("MappingsFile factory failed, expected redirect uri: [%s], got [%s]", redirect, mapping.Redirect)
	}
}

func Test_MappingValidate(t *testing.T) {
	path := "/"
	redirect := "https://127.0.0.1"
	mapping := NewMapping(path, redirect)

	_, err := mapping.Validate()
	if err != nil {
		t.Errorf("Could not parse and validate new MappingsFile, error:[%s]", err)
	}
}

/**
Here we go one level deeper than above's validate to test the scheme. We mutate and force
all schemes to be HTTPS. Test to make sure this actually happens.
 */
func Test_MappingScheme(t *testing.T) {
	expectedScheme := "https"

	path := "/"
	redirect := "https://127.0.0.1"
	mapping := NewMapping(path, redirect)

	url, err := mapping.Validate()
	if err != nil {
		t.Errorf("Could not parse and validate new MappingsFile, error:[%s]", err)
	} else {
		if url.Scheme != expectedScheme {
			t.Errorf("Expected scheme [%s], but got [%s]", expectedScheme, url.Scheme)
		}
	}
}

func Test_badMappings(t *testing.T) {
	for index, testData := range badMappings {
		mappingEntry := NewMapping(testData.path, testData.redirect)
		if _, err := mappingEntry.Validate(); err == nil {
			msg := fmt.Sprintf("Expected badMappings[%d] to be invalid, ended up being valid.", index)
			t.Errorf(msg)
		}
	}
}

/**
Here we test access to the mappings map. We also enforce that it is a map if anyone changes it.
 */
func Test_MappingsMap(t *testing.T) {
	expectedKey := "test"

	redirectMap := MappingsFile{
		Mappings: map[string]Mapping{
			expectedKey: Mapping{"/mypath", "https://127.0.0.1"},
		},
	}

	// Get something we know exists
	if _, err := redirectMap.Get(expectedKey); err != nil {
		t.Errorf("Expected a mapping")
	}

	// Get a key that does not exist
	if _, err := redirectMap.Get("n/a"); err == nil {
		t.Errorf("Expected to get an error for key[%s]", "n/a")
	}
}
