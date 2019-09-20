package azurestack

import (
	"encoding/json"
	"errors"
	"github.com/iancoleman/strcase"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const deploymentARMTemplate = "template.json"
const funcFile = "function.json"
const funcMemSize = "1536"

type functionInfo struct {
	Disabled bool `json:"disabled"`
	Bindings []struct {
		Type      string `json:"type"`
		Direction string `json:"direction"`
		Name      string `json:"name"`
		AuthLevel string `json:"authLevel,omitempty"`
	} `json:"bindings"`
	EntryPoint string `json:"entryPoint"`
	ScriptFile string `json:"scriptFile"`
}

type AzureDeployment struct {
	path            string
	functionsPath   string
	functionAppName string
	name            string
	location        string
	runtime         string
	group           string
	nodeVer         string
	Functions       []*AzureFunction
	deployed        bool
}

func getSubdirectories(path string) ([]os.FileInfo, error) {
	var subdirectories []os.FileInfo
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			subdirectories = append(subdirectories, file)
		}
	}
	return subdirectories, nil
}

func (d *AzureDeployment) Deploy() error {
	var err error
	if d.deployed {
		return errors.New("deployment is already deployed")
	}
	d.setDeploymentEpoch()
	_, _, err = ExecCmd(d.path, "az", "group", "create", "-n", d.group, "-l", d.location)
	if err != nil {
		return err
	}
	d.deployed = true
	err = d.deployFunctionApp()
	if err != nil {
		return err
	}
	err = d.deployFunctions()
	retries := 10
	for err != nil && retries > 0 {
		time.Sleep(5 * time.Second)
		err = d.deployFunctions()
		retries--
	}
	if err != nil {
		return err
	}
	return nil
}

func (d *AzureDeployment) deployFunctions() error {
	_, _, err := ExecCmd(d.functionsPath, "func", "azure", "functionapp", "publish", d.functionAppName)
	return err
}

func (d *AzureDeployment) setDeploymentEpoch() {
	d.functionAppName = d.name + strconv.FormatInt(time.Now().UnixNano(), 10)
	for _, function := range d.Functions {
		function.functionAppName = d.functionAppName
	}
	d.group = d.functionAppName + "-faastestrg"
}

func (d *AzureDeployment) deployFunctionApp() error {
	cmd := []string{
		"group", "deployment", "create", "--template-file", deploymentARMTemplate, "-g", d.group, "--parameters"}
	parameters := []string{"functionAppName=" + d.functionAppName, "location=" + d.location}
	if d.nodeVer != "" {
		parameters = append(parameters, "nodeVersion="+d.nodeVer)
	}
	cmd = append(cmd, parameters...)
	_, _, err := ExecCmd(d.path, "az", cmd...)
	if err != nil {
		return err
	}
	return nil
}

func (d *AzureDeployment) listFunctions() ([]*AzureFunction, error) {
	var functions []*AzureFunction
	var funcInfo functionInfo
	subdirectories, err := getSubdirectories(d.functionsPath)
	if err != nil {
		return nil, err
	}
	for _, dir := range subdirectories {
		funcPath := filepath.Join(d.functionsPath, dir.Name())
		rawFuncInfo, err := ioutil.ReadFile(filepath.Join(funcPath, funcFile))
		if err != nil {
			continue
		}
		err = json.Unmarshal(rawFuncInfo, &funcInfo)
		if err != nil {
			return nil, err
		}
		functions = append(functions, &AzureFunction{
			name:       strcase.ToCamel(dir.Name()),
			handler:    funcInfo.EntryPoint,
			memorySize: funcMemSize,
			runtime:    d.runtime,
		})
	}
	return functions, nil
}

func (d *AzureDeployment) Remove() error {
	_, _, err := ExecCmd(d.path, "az", "group", "delete", "-n", d.group, "-y", "--no-wait")
	if err != nil {
		return err
	}
	d.deployed = false
	return nil
}
