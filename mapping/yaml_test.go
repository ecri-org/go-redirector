package mapping

import (
	"fmt"
	"reflect"
	"testing"
)

func newEntry(immediate bool, redirect string) Entry {
	return Entry{
		immediate,
		redirect,
	}
}

/**
These patterns will not pass validation.
*/
var badMappings = []struct {
	path         string
	mappingEntry Entry
}{
	{
		"", // empty path
		newEntry(
			true,
			"https://127.0.0.1",
		),
	},
	{
		"pathA",
		newEntry(
			true,
			"://127.0.0.1", // no scheme
		),
	},
	{
		"pathA", // path has no slash prefix
		newEntry(
			true,
			"https://127.0.0.1",
		),
	},
	{
		"/pathA",
		newEntry(
			true,
			"http://127.0.0.1", // we only accept https, sorry
		),
	},
	{
		"/pathA",
		newEntry(
			true,
			"ftp://127.0.0.1", // we only accept https, sorry
		),
	},
	{
		"/pathA",
		newEntry(
			true,
			"ftp//127.0.0.1", // bad URI
		),
	},
	{
		"/pathA",
		newEntry(
			true,
			"ftp//127.0.0./?", // bad path
		),
	},
	{
		"/pathA#fragment",
		newEntry(
			true,
			"https://\u007F", // rune in path
		),
	},
	{
		"/\x7f#fragment", // rune in path
		newEntry(
			true,
			"https://127.0.0.1",
		),
	},
	{
		"", // empty path
		newEntry(
			true,
			"https://127.0.0.1",
		),
	},
}

func Test_MappingValidate(t *testing.T) {
	mapping := Mapping{
		"/": newEntry(true, "https://127.0.0.1"),
	}

	// Test valid
	if err := mapping.Validate(); err != nil {
		t.Errorf("Could not parse and validate new MappingsFile, error:[%s]", err)
	}
}

func Test_MappingScheme(t *testing.T) {
	mapping := Mapping{
		"/": newEntry(true, "https://127.0.0.1"),
	}

	if err := mapping.Validate(); err != nil {
		t.Errorf("Could not parse and validate new MappingsFile, error:[%s]", err)
	}
}

func Test_badMappings(t *testing.T) {
	for index, testData := range badMappings {
		mapping := Mapping{
			testData.path: Entry{
				Immediate: testData.mappingEntry.Immediate,
				Redirect:  testData.mappingEntry.Redirect,
			},
		}
		if err := mapping.Validate(); err == nil {
			msg := fmt.Sprintf("Expected badMappings[%d] to be invalid, ended up being valid.\n", index)
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
		Mappings: map[string]*Mapping{
			expectedKey: {
				"/mypath":  newEntry(true, "https://127.0.0.1"),
				"/mypath2": newEntry(true, "https://127.0.0.1"),
			},
		},
	}

	// GetRedirectURI something we know exists
	if value := redirectMap.GetRedirectURI(expectedKey, "/mypath"); value == "" {
		t.Errorf("Expected a mapping")
	}

	// GetRedirectURI a key that does not exist
	if value := redirectMap.GetRedirectURI("n/a", ""); value != "" {
		t.Errorf("Expected to get an error for a search of key[%s]", "n/a")
	}
}

func Test_EmptyMappingFile(t *testing.T) {
	testFile := `---
`

	if _, err := Parse([]byte(testFile)); err == nil {
		t.Errorf("Expect to get an error with an empty yaml mapping file.")
	}
}

func Test_EmptyMappingListing(t *testing.T) {
	testFile := `---
mapping:
`

	if _, err := Parse([]byte(testFile)); err == nil {
		t.Errorf("Expect an error with a mapping file with no entries.")
	}
}

func Test_MappingFileWithLocalhost(t *testing.T) {
	testFile := `---
mapping:
  localhost:
    "/my-path":
      immediate: false
      redirect: https://localhost:8081
    "/":
      redirect: https://localhost:8082
`

	if _, err := Parse([]byte(testFile)); err == nil {
		t.Errorf("Data was expected to be invalid as you cannot use localhost: %v", err)
	}
}

func Test_MappingFileWithRoot(t *testing.T) {
	testFile := `---
mapping:
  testhost:
    "/my-path":
      immediate: false
      redirect: https://localhost:8081
    "/":
      redirect: https://localhost:8082
`

	if data, err := Parse([]byte(testFile)); err != nil {
		t.Errorf("Data was expected to be valid: %v", err)
	} else {

		if uri := data.GetRedirectURI("testhost", "/my-path"); uri != "https://localhost:8081" {
			t.Errorf("Incorrect URI obtained, expected https://localhost:8081, got [%s]", uri)
		}

		if uri := data.GetRedirectURI("testhost", "/"); uri != "https://localhost:8082" {
			t.Errorf("Incorrect URI obtained, expected https://localhost:8082, got [%s]", uri)
		}

		// we treat root as a wildcard pattern
		if uri := data.GetRedirectURI("testhost", "/something-not-there"); uri != "https://localhost:8082" {
			t.Errorf("Incorrect URI obtained, expected https://localhost:8082, got [%s]", uri)
		}
	}
}

func Test_MappingFileWithoutRoot(t *testing.T) {
	testFile := `---
mapping:
  testhost:
    "/my-path":
      redirect: https://localhost:8081
`
	if data, err := Parse([]byte(testFile)); err != nil {
		t.Errorf("Could not parse test data: %v", err)
	} else {
		if err := data.Validate(); err != nil {
			t.Errorf("Data was expected to be valid: %v", err)
		}

		if uri := data.GetRedirectURI("testhost", "/my-path"); uri != "https://localhost:8081" {
			t.Error("Incorrect URI obtained, expected https://localhost:8081")
		}

		if uri := data.GetRedirectURI("testhost", "/"); uri != "" {
			t.Error("Incorrect URI obtained, expected empty string since mapping doesn't specify a wildcard root '/'")
		}
	}
}

func Test_MappingFileWithEmptyPath(t *testing.T) {
	redirectMap := MappingsFile{
		Mappings: map[string]*Mapping{
			"some-test-host": {
				"":         newEntry(true, "https://127.0.0.1"),
				"/mypath2": newEntry(true, "https://127.0.0.1"),
			},
		},
	}

	if err := redirectMap.Validate(); err == nil {
		t.Errorf("Expected to see an error with the path being empty")
	}
}

/**
Rely on the tests above to test the mapping. Here we test for files that exist, or those
that cannot be loaded via `yaml.Unmarshal()`.
*/
func Test_LoadMappingFile(t *testing.T) {
	testFile := "../tests/test-redirect-map.yml"
	missingFile := "../tests/noop.yml"
	badFile := "../tests/bad-redirect-map.yml"

	// Load file which does not exist
	if _, err := LoadMappingFile(missingFile); err == nil {
		t.Errorf("Expected to see an error when using a missing file [%s].", missingFile)
	}

	// Load bad file, should yield 'cannot unmarshal'
	if _, err := LoadMappingFile(badFile); err == nil {
		t.Errorf("Expected to see an error when trying to parse a bad file [%s].", badFile)
	}

	// Test real file
	if file, err := LoadMappingFile(testFile); err != nil {
		t.Errorf("Expected to find the test Redirect map yaml file [%s] and parse it.", err)
	} else {
		if err := file.Validate(); err != nil {
			t.Errorf("Test failed as could not validate test file, see error %s", err)
		}

		keys := reflect.ValueOf(file.Mappings).MapKeys()
		if len(keys) != 1 {
			t.Errorf("Expected test file to have size of 1")
		}

		mapping := file.Mappings[keys[0].String()]
		if len(*mapping) != 3 { // i.e. how many path entries exist for the host
			t.Errorf("Expected to find two path mappings for the key [%s], instead found [%d]", keys[0], len(*mapping))
		}
	}
}

func Test_GetMappingEntryNoRoot(t *testing.T) {
	host := "testhost"
	testFile := fmt.Sprintf(`---
mapping:
  %s:
    "/my-path":
      redirect: https://localhost:8081
`, host)

	mappingsFile, err := Parse([]byte(testFile))
	if err != nil {
		t.Errorf("Data was expected to be valid: %v", err)
	}

	// found entry
	path := "/my-path"
	if _, entryError := mappingsFile.GetMappingEntry(host, path); entryError != nil {
		t.Errorf("Expected to be able and retreive path: [%s]", path)
	}

	// invalid entry
	path = "/some-other-path"
	if _, entryError := mappingsFile.GetMappingEntry(host, path); entryError == nil {
		t.Errorf("Expected not to be able and retreive path: [%s], error: [%s]", path, entryError)
	}
}

func Test_GetMappingEntryWithRoot(t *testing.T) {
	host := "testhost"
	testFile := fmt.Sprintf(`---
mapping:
  %s:
    "/my-path":
      redirect: https://localhost:8081
    "/":
      redirect: https://localhost:8082
`, host)

	mappingsFile, err := Parse([]byte(testFile))
	if err != nil {
		t.Errorf("Data was expected to be valid: %v", err)
	}

	// found entry
	path := "/my-path"
	if _, entryError := mappingsFile.GetMappingEntry(host, path); entryError != nil {
		t.Errorf("Expected to be able and retreive path: [%s]", path)
	}

	// invalid entry
	path = "/some-other-path"
	if _, entryError := mappingsFile.GetMappingEntry(host, path); entryError != nil {
		t.Errorf("Expected to see root as that is the fall through wildcard path when looking for path: [%s], error: [%s]", path, entryError)
	}
}

func Test_GetMappingEntryWithWildcard(t *testing.T) {
	host := "testhost"
	testFile := fmt.Sprintf(`---
mapping:
  %s:
    "/my-path":
      redirect: https://localhost:8081
    "*":
      redirect: https://localhost:8082
`, host)

	mappingsFile, err := Parse([]byte(testFile))
	if err != nil {
		t.Errorf("Data was expected to be valid: %v", err)
	}

	// found entry
	path := "/my-path"
	if _, entryError := mappingsFile.GetMappingEntry(host, path); entryError != nil {
		t.Errorf("Expected to be able and retreive path: [%s]", path)
	}

	// invalid entry
	path = "/some-other-path"
	if _, entryError := mappingsFile.GetMappingEntry(host, path); entryError != nil {
		t.Errorf("Expected to see root as that is the fall through wildcard path when looking for path: [%s], error: [%s]", path, entryError)
	}
}
