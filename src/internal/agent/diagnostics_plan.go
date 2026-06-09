package agent

import (
	"fmt"
	"strings"
)

// ClusterDiagnosticsPlan is the deep Module for cluster-side diagnostics
// evidence. Its Interface exposes named kubectl evidence steps and pod-log
// fallback sections without making the collector know the Dynamo labels,
// section names, or command shapes.
type ClusterDiagnosticsPlan struct {
	Resource ResourceRef
}

type DiagnosticsCommand struct {
	Name string
	Args []string
}

func NewClusterDiagnosticsPlan(ref ResourceRef) ClusterDiagnosticsPlan {
	return ClusterDiagnosticsPlan{Resource: ref}
}

func (p ClusterDiagnosticsPlan) EvidenceCommands(tailLines int) []DiagnosticsCommand {
	ref := p.Resource
	selector := servingApplicationSelector(ref.Name)
	dynamoSelector := dynamoGraphDeploymentSelector(ref.Name)
	return []DiagnosticsCommand{
		{Name: "dynamographdeployment", Args: []string{"-n", ref.Namespace, "get", "dynamographdeployment", ref.Name, "-o", "yaml"}},
		{Name: "dynamocomponentdeploymentsByLabel", Args: []string{"-n", ref.Namespace, "get", "dynamocomponentdeployment", "-l", selector, "-o", "wide"}},
		{Name: "dynamocomponentdeploymentByName", Args: []string{"-n", ref.Namespace, "get", "dynamocomponentdeployment", ref.Name, "-o", "wide"}},
		{Name: "podsByLabel", Args: []string{"-n", ref.Namespace, "get", "pod", "-l", selector, "-o", "wide"}},
		{Name: "podsByDynamoLabel", Args: []string{"-n", ref.Namespace, "get", "pod", "-l", dynamoSelector, "-o", "wide"}},
		{Name: "podsByNamePrefix", Args: []string{"-n", ref.Namespace, "get", "pod", "-o", "name"}},
		{Name: "events", Args: []string{"-n", ref.Namespace, "get", "events", "--sort-by=.lastTimestamp"}},
		{Name: "currentLogsByLabel", Args: labelLogArgs(ref.Namespace, selector, false, tailLines)},
		{Name: "previousLogsByLabel", Args: labelLogArgs(ref.Namespace, selector, true, tailLines)},
	}
}

func (p ClusterDiagnosticsPlan) PodLogCommands(podNames []string, previous bool, tailLines int) []DiagnosticsCommand {
	commands := make([]DiagnosticsCommand, 0, len(podNames))
	for _, podName := range podNames {
		args := []string{"-n", p.Resource.Namespace, "logs", podName, "--all-containers=true"}
		if previous {
			args = append(args, "--previous")
		}
		args = append(args, "--tail", fmt.Sprintf("%d", tailLines), "--prefix=true")
		commands = append(commands, DiagnosticsCommand{Name: podName, Args: args})
	}
	return commands
}

func (p ClusterDiagnosticsPlan) PodLogSectionName(previous bool) string {
	if previous {
		return "previousLogsByNamePrefix"
	}
	return "currentLogsByNamePrefix"
}

func servingApplicationSelector(resourceName string) string {
	return "inference.zhiliu.dev/serving-application=" + resourceName
}

func dynamoGraphDeploymentSelector(resourceName string) string {
	return "nvidia.com/dynamo-graph-deployment-name=" + resourceName
}

func labelLogArgs(namespace string, selector string, previous bool, tailLines int) []string {
	args := []string{"-n", namespace, "logs", "-l", selector, "--all-containers=true"}
	if previous {
		args = append(args, "--previous")
	}
	return append(args, "--tail", fmt.Sprintf("%d", tailLines), "--prefix=true")
}

func podNamesByPrefix(output string, prefix string) []string {
	var podNames []string
	for _, line := range strings.Split(output, "\n") {
		name := strings.TrimSpace(line)
		name = strings.TrimPrefix(name, "pod/")
		if name == "" || !strings.HasPrefix(name, prefix) {
			continue
		}
		podNames = append(podNames, name)
	}
	return podNames
}
