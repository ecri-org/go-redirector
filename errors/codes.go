package errors

//type errorType int

const (
	// ExitCodeConfigError defines an error with the app configuration
	ExitCodeConfigError int = iota
	// ExitCodeExecutionFailure defines an error with the execution of the app
	ExitCodeExecutionFailure
	// ExitCodeAppDevError defines an error with the execution of the app
	ExitCodeAppDevError
	// ExitCodeBadPort defines an error with a bad port
	ExitCodeBadPort
	// ExitCodeTemplateFilenameEmpty defines an error when the template file name specified was empty
	ExitCodeTemplateFilenameEmpty
	// ExitCodeTplNotFound defines an error when the template could not be found
	ExitCodeTplNotFound
	// ExitCodeTplError defines an error with the template
	ExitCodeTplError
	// ExitCodeBadMappingFile defines an error when the mapping file is bad, cannot be parsed or has syntax issues
	ExitCodeBadMappingFile
	// ExitCodeInvalidLoglevel defines an error when an invalid log level is used
	ExitCodeInvalidLoglevel
	// ExitMetricsIssue defines an error when there is an issue with the Metrics endpoint
	ExitMetricsIssue
)
