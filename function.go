package azurestack

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"net/url"
	"strings"
)

type AzureFunction struct {
	name            string
	handler         string
	runtime         string
	memorySize      string
	functionAppName string
}

func (f *AzureFunction) Name() string {
	return strcase.ToKebab(f.name)
}

func (f *AzureFunction) Handler() string {
	return f.handler
}

func (f *AzureFunction) Description() string {
	return strcase.ToDelimited(f.name, ' ')
}

func (f *AzureFunction) Runtime() string {
	if strings.HasPrefix("node8", f.runtime) || strings.HasPrefix("nodejs8", f.runtime) {
		return "nodejs8"
	}
	if strings.HasPrefix("node10", f.runtime) || strings.HasPrefix("nodejs10", f.runtime) {
		return "nodejs10"
	}
	return f.runtime
}

func (f *AzureFunction) MemorySize() string {
	return f.memorySize
}

func (f *AzureFunction) InvokeURL() url.URL {
	var retval url.URL
	retval.Scheme = "https"
	retval.Host = fmt.Sprintf("%s.azurewebsites.net", f.functionAppName)
	retval.Path = fmt.Sprintf("/api/%s", f.name)
	return retval
}
