package errors

import (
	"testing"
)

func Test_ErrorsType(t *testing.T) {
	codes := []int{
		ExitCodeConfigError,
		ExitCodeExecutionFailure,
		ExitCodeAppDevError,
		ExitCodeBadPort,
		ExitCodeTemplateFilenameEmpty,
		ExitCodeTplNotFound,
		ExitCodeTplError,
		ExitCodeBadMappingFile,
		ExitCodeInvalidLoglevel,
		ExitMetricsIssue,
	}

	for code := range codes {
		if code < 0 {
			t.Errorf("Invalid code, not of type Int")
		}
	}
}
