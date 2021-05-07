package mapping

import (
	"fmt"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
)

// Mapping is a type which is used to store mapping in the mappings file
type Mapping map[string]string

// Get an entry from the mapping
func (m Mapping) Get(entry string) string {
	if value, ok := m[entry]; ok {
		return value
	}
	return ""
}

// Validate a single mapping
func (m Mapping) Validate() error {
	for path, redirectURI := range m {
		if path == "" {
			msg := "Found empty string as path."
			log.Errorf(msg)
			return errors.New(msg)
		}
		if path[0] != '/' {
			msg := fmt.Sprintf("Redirect uri [%s] must always be prefixed with '/', no relative paths accepted here.\n", path)
			log.Errorf(msg)
			return errors.New(msg)
		}
		if _, err := url.ParseRequestURI(path); err != nil {
			return err
		}

		uri, err := url.ParseRequestURI(redirectURI)
		if err != nil {
			log.Debugf("Redirect uri is not fully qualified.")
			return err
		}

		if uri.Scheme != "https" {
			msg := fmt.Sprintf("Redirect uri scheme on [%s] needs to be changed and use 'https' as the scheme.", uri.String())
			return errors.New(msg)
		}

		log.Debugf("Parsed %s", uri.String())
	}

	return nil
}

// MappingsFile describes the mapping file
type MappingsFile struct {
	Mappings map[string]Mapping `yaml:"mapping,omitempty"`
}

// NewMappingsFile is a factory which creates new mappings file.
func NewMappingsFile() *MappingsFile {
	return &MappingsFile{}
}

// Validate validates the mappings file entirely
func (m *MappingsFile) Validate() error {
	if len(m.Mappings) == 0 {
		return errors.New("Mapping file is empty or has no entries, please provide some")
	}
	for host, entry := range m.Mappings {
		if host == "localhost" {
			return errors.New("Localhost is reserved, you cannot use this host")
		}
		if err := entry.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GetRedirectURI gets the URI of a matching host and path from the mappings file
func (m *MappingsFile) GetRedirectURI(host string, path string) string {
	if mappingEntry, ok := m.Mappings[host]; ok {
		// look for specific
		if uri := mappingEntry.Get(path); uri != "" {
			return uri
		}

		// look for root TODO: might be better to sort later
		if uri := mappingEntry.Get("/"); uri != "" {
			return uri
		}
	}

	msg := fmt.Sprintf("Could not find host and path [%s%s]", host, path)
	log.Debugf(msg)
	return ""
}

// Parse the mapping file.
func Parse(data []byte) (*MappingsFile, error) {
	mappingFile := NewMappingsFile()

	if err := yaml.Unmarshal([]byte(data), mappingFile); err != nil {
		return mappingFile, err
	}

	if err := mappingFile.Validate(); err != nil {
		return mappingFile, err
	}

	return mappingFile, nil
}

// LoadMappingFile loads a file, assuming it is a redirect map file.
func LoadMappingFile(file string) (*MappingsFile, error) {
	data, err := ioutil.ReadFile(file)

	if err != nil {
		msg := fmt.Sprintf("Could not find file: %s", file)
		return nil, errors.Errorf(msg)
	}

	log.Debug("Able to parse yaml file")
	return Parse(data)
}
