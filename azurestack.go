package azurestack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const deploymentSettingsFile = "local.settings.json"
const stackConfigFile = "stack.json"
const NodeRuntime = "node"
const dotnetRuntime = "dotnet"
const javaRuntime = "java"

type deploymentInfo struct {
	IsEncrypted bool `json:"IsEncrypted"`
	Values      struct {
		Runtime             string `json:"FUNCTIONS_WORKER_RUNTIME"`
		NodeVer             string `json:"WEBSITE_NODE_DEFAULT_VERSION"`
		AzureWebJobsStorage string `json:"AzureWebJobsStorage"`
	} `json:"Values"`
}

type stackInfo struct {
	Name     string `json:"name"`
	Location string `json:"location"`
	Project  string `json:"project"`
	Stage    string `json:"stage"`
}

type AzureStack struct {
	stackInfo
	path        string
	deployments []*AzureDeployment
	Functions   []*AzureFunction
}

func New(path string) (*AzureStack, error) {
	var info stackInfo
	stackInfoPath := filepath.Join(path, stackConfigFile)
	rawStackInfo, err := ioutil.ReadFile(stackInfoPath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(rawStackInfo, &info)
	if err != nil {
		return nil, err
	}
	stack := AzureStack{stackInfo: info, path: path}
	err = stack.listDeployments()
	if err != nil {
		return nil, err
	}
	for _, deployment := range stack.deployments {
		stack.Functions = append(stack.Functions, deployment.Functions...)
	}
	return &stack, nil
}

func (s *AzureStack) newDeployment(path string) error {
	var info deploymentInfo
	_, deploymentDir := filepath.Split(path)
	path = filepath.Clean(path) + string(os.PathSeparator)
	deploymentInfoPath := filepath.Join(path, deploymentSettingsFile)
	rawDeploymentInfo, err := ioutil.ReadFile(deploymentInfoPath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(rawDeploymentInfo, &info)
	if err != nil {
		return err
	}
	runtime := info.Values.Runtime
	if runtime == NodeRuntime && info.Values.NodeVer != "" {
		runtime += info.Values.NodeVer
	}
	deployment := AzureDeployment{
		path:          path,
		functionsPath: path,
		runtime:       runtime,
		location:      s.Location,
		nodeVer:       info.Values.NodeVer,
		deployed:      false,
	}
	deployment.name = s.Name + deploymentDir
	if runtime == dotnetRuntime {
		_, _, err = ExecCmd(deployment.path, "dotnet", "build", "--output", "bin/publish")
		if err != nil {
			return err
		}
		deployment.functionsPath = filepath.Join(deployment.path, "bin", "publish")
	}
	if runtime == javaRuntime {
		_, _, err = ExecCmd(deployment.path, "mvn", "package")
		if err != nil {
			return err
		}
		deployment.functionsPath = filepath.Join(deployment.path, "target", "azure-functions", "deployment")
	}
	deployment.Functions, err = deployment.listFunctions()
	if err != nil {
		return err
	}
	s.deployments = append(s.deployments, &deployment)
	return nil
}

func (s *AzureStack) DeployStack() error {
	for _, deployment := range s.deployments {
		err := deployment.Deploy()
		if err == nil {
			continue
		}
		// stack deployment failed, cleaning up by removing whatever was partially deployed:
		cleanupErr := s.RemoveStack()
		if cleanupErr != nil {
			return fmt.Errorf("deploy error: %s, cleanup error: %s", err.Error(), cleanupErr.Error())
		}
		return err
	}
	return nil
}

func (s *AzureStack) RemoveStack() error {
	for _, deployment := range s.deployments {
		if !deployment.deployed {
			continue
		}
		err := deployment.Remove()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *AzureStack) StackId() string {
	return s.Name
}

func (s *AzureStack) Project() string {
	return s.stackInfo.Project
}

func (s *AzureStack) Stage() string {
	return s.stackInfo.Stage
}

func (s *AzureStack) listDeployments() error {
	subdirectories, err := getSubdirectories(s.path)
	if err != nil {
		return err
	}
	for _, dir := range subdirectories {
		deploymentPath := filepath.Join(s.path, dir.Name())
		_, err := os.Stat(filepath.Join(deploymentPath, deploymentARMTemplate))
		if err != nil {
			continue
		}
		err = s.newDeployment(deploymentPath)
		if err != nil {
			return err
		}
	}
	return nil
}
