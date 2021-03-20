package mapping

import (
	"fmt"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
)

type Mapping struct {
	Path string `yaml:"path"`
	Redirect string `yaml:"redirect"`
}

/**
  This validator is weak. I also did not want to do any redirect host lookups
  as I expect some to be down. As such that should not fail the start of this
  application.
  This author chooses to enforce https redirects, even if the mapping says http,
  telnet, or whatever protocol is used.
  Sorry. See: https://www.eff.org/https-everywhere
 */
func (m *Mapping) Validate() (*url.URL, error) {
	defaultScheme := "https"
	//fqUri := fmt.Sprintf("%s%s", defaultScheme, m.Path)

	if m.Path == "" {
		msg := fmt.Sprintf("Redirect uri must not be empty. Use '/' if root is desired.")
		log.Errorf(msg)
		return nil, errors.New(msg)
	}

	if m.Path[0] != '/' {
		msg := fmt.Sprintf("Redirect uri [%s] must always be prefixed with '/', no relative paths accepted here.", m.Path)
		log.Errorf(msg)
		return nil, errors.New(msg)
	}

	if _, err := url.ParseRequestURI(m.Path); err != nil {
		return nil, err
	}

	uri, err := url.ParseRequestURI(m.Redirect)
	if err != nil {
		log.Debugf("Redirect uri is not fully qualified.")
		return nil, err
	}

	if uri.Scheme != "https" {
		msg := fmt.Sprintf("Redirect uri scheme on [%s] needs to change and use 'https' as the scheme.", uri.String())
		return nil, errors.New(msg)
	}

	uri.Scheme = defaultScheme

	log.Debugf("Parsed %s", uri.String())

	return uri, nil
}

/**
Factory
 */
func NewMapping(path string, redirect string) *Mapping {
	return &Mapping{path, redirect}
}

/**
Since we only expect to read from this map, I'm satisified with a pointer
receiver. If ever we need to sync, we should change this to a value receiver
for maximum safety.
NOTE: concurrent map write see above
 */
type MappingsFile struct {
	Mappings map[string]Mapping `yaml:"mapping,omitempty"`
}

func (m *MappingsFile) Validate() error {
	for _, entry := range m.Mappings {
		if _, err := entry.Validate(); err != nil {
			return err
		}
	}

	return nil
}

/**
We take special care not to return a pointer for Mapping for
maximum safety. This means if ever we expected to write it won't
work until we change this, and also change the value reciever for
Mapping type as well.
NOTE: concurrent map write see above
 */
func (m *MappingsFile) Get(key string) (Mapping, error) {
	if value, ok := m.Mappings[key]; ok {
		return value, nil
	}

	return Mapping{}, errors.New(fmt.Sprintf("Could not find key [%s]", key))
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