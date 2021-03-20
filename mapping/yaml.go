package mapping

import (
	"fmt"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
)

type Mapping map[string]string

func (m Mapping) Get(entry string) (string) {
	if value, ok := m[entry]; ok {
		return value
	}
	return ""
}

func (m Mapping) Validate() error {
	for path, redirectUri := range m {
		if path == "" {
			msg := fmt.Sprintf("Found empty string as path.")
			log.Errorf(msg)
			return errors.New(msg)
		}
		if path[0] != '/' {
			msg := fmt.Sprintf("Redirect uri [%s] must always be prefixed with '/', no relative paths accepted here.", path)
			log.Errorf(msg)
			return errors.New(msg)
		}
		if _, err := url.ParseRequestURI(path); err != nil {
			return err
		}

		uri, err := url.ParseRequestURI(redirectUri)
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

type MappingsFile struct {
	Mappings map[string]Mapping `yaml:"mapping,omitempty"`
}

func (m *MappingsFile) Validate() error {
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

func (m *MappingsFile) GetRedirectUri(host string, path string) string {
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

/**
Parse the mapping file.
 */
func Parse(data []byte) (*MappingsFile, error) {
	mappingFile := MappingsFile{}

	if err := yaml.Unmarshal([]byte(data), &mappingFile); err !=nil {
		return &mappingFile, err
	}

	if err := mappingFile.Validate(); err != nil {
		return &mappingFile, err
	}

	return &mappingFile, nil
}

/**
Load a file, assuming it is a redirect map file.
 */
func LoadMappingFile(file string) (*MappingsFile, error) {
	if data, err := ioutil.ReadFile(file); err != nil {
		msg := fmt.Sprintf("Could not find file: %s", file)
		return nil, errors.Errorf(msg)
	} else {
		log.Debug("Able to parse yaml file")
		return Parse(data)
	}
}