package mapping

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
)

// Entry defines the inner object for each path
type Entry struct {
	Immediate bool   `yaml:"immediate,omitempty"`
	Redirect  string `yaml:"redirect,omitempty"`
}

// Mapping is a type which is used to store mapping in the mappings file
type Mapping map[string]Entry

// Get an entry from the mapping
func (m Mapping) Get(entry string) Entry {
	return m[entry]
}

func validStart(path string) bool {
	if path == "" {
		return false
	}

	if path[0] == '/' {
		return true
	}

	if path[0] == '*' {
		return true
	}

	return false
}

// Validate a single mapping
func (m *Mapping) Validate() error {
	logEntry := func(entry *Entry, path string) {
		isFriendly := entry.Immediate
		if isFriendly {
			log.Debug().Msg(fmt.Sprintf("Evaluating friendly redirect from path [%s] to [%s]", path, entry.Redirect))
		} else {
			log.Debug().Msg(fmt.Sprintf("Evaluating direct redirect from path [%s] to [%s]", path, entry.Redirect))
		}
	}

	// WARNING: only return errors, leaving the last line a fall through nil. An early return
	//          will cause issues as file reads are async you risk exiting validation earlier
	//          than intended.
	validate := func(entry *Entry, path string) error {
		if !validStart(path) {
			msg := fmt.Sprintf("Redirect uri [%s] must always be prefixed with '/' or '*', no relative or empty paths accepted here.", path)
			log.Error().Msg(msg)
			return errors.New(msg)
		}

		if _, err := url.ParseRequestURI(path); err != nil {
			return err
		}

		uri, err := url.ParseRequestURI(entry.Redirect)
		if err != nil {
			log.Debug().Msg("Redirect uri is not fully qualified.")
			return err
		}

		if uri.Scheme != "https" {
			msg := fmt.Sprintf("Redirect uri scheme on [%s] needs to be changed and use 'https' as the scheme.", uri.String())
			return errors.New(msg)
		}

		return nil
	}

	for path := range *m {
		entry := m.Get(path)
		logEntry(&entry, path)
		if err := validate(&entry, path); err != nil {
			return err
		}

	}

	return nil
}

// MappingsFile describes the mapping file
type MappingsFile struct {
	Mappings map[string]*Mapping `yaml:"mapping,omitempty"`
}

// NewMappingsFile is a factory which creates new mappings file.
func NewMappingsFile() *MappingsFile {
	return &MappingsFile{
		Mappings: map[string]*Mapping{},
	}
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
		if entry := mappingEntry.Get(path); entry.Redirect != "" {
			return entry.Redirect
		}

		// look for root TODO: might be better to sort later
		if entry := mappingEntry.Get("/"); entry.Redirect != "" {
			return entry.Redirect
		}
	}

	msg := fmt.Sprintf("Could not find host and path [%s%s]", host, path)
	log.Debug().Msg(msg)
	return ""
}

// GetMappingEntry returns an entry for a particular mapping given the user defined host and path
func (m *MappingsFile) GetMappingEntry(host string, path string) (*Entry, error) {
	if mappingEntry, ok := m.Mappings[host]; ok {
		// look for specific
		if entry := mappingEntry.Get(path); entry.Redirect != "" {
			return &entry, nil
		}

		// look for root TODO: might be better to sort later
		if entry := mappingEntry.Get("/"); entry.Redirect != "" {
			return &entry, nil
		}

		// look for wildcard
		if entry := mappingEntry.Get("*"); entry.Redirect != "" {
			return &entry, nil
		}
	}

	msg := fmt.Sprintf("Could not find host and path [%s%s]", host, path)
	log.Debug().Msg(msg)
	return nil, errors.New(msg)
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

// LoadMappingFile loads a file, assuming it is a Redirect map file.
func LoadMappingFile(file string) (*MappingsFile, error) {
	data, err := ioutil.ReadFile(file)

	if err != nil {
		msg := fmt.Sprintf("Could not find file: %s", file)
		return nil, errors.Errorf(msg)
	}

	log.Debug().Msg("Able to parse yaml file")
	return Parse(data)
}
