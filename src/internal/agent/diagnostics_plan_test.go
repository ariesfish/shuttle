package agent

import (
	"strings"
	"testing"
)

func TestClusterDiagnosticsPlanDefinesEvidenceAndPodLogFallback(t *testing.T) {
	plan := NewClusterDiagnosticsPlan(ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"})

	commands := plan.EvidenceCommands(50)
	if len(commands) != 9 {
		t.Fatalf("expected evidence commands, got %+v", commands)
	}
	assertPlanCommand(t, commands, "dynamographdeployment", "get dynamographdeployment deepseek-v4-flash -o yaml")
	assertPlanCommand(t, commands, "podsByLabel", "get pod -l inference.zhiliu.dev/serving-application=deepseek-v4-flash -o wide")
	assertPlanCommand(t, commands, "podsByDynamoLabel", "get pod -l nvidia.com/dynamo-graph-deployment-name=deepseek-v4-flash -o wide")
	assertPlanCommand(t, commands, "previousLogsByLabel", "logs -l inference.zhiliu.dev/serving-application=deepseek-v4-flash --all-containers=true --previous --tail 50")

	podLogs := plan.PodLogCommands([]string{"deepseek-v4-flash-abc"}, true, 25)
	if len(podLogs) != 1 || podLogs[0].Name != "deepseek-v4-flash-abc" || !strings.Contains(strings.Join(podLogs[0].Args, " "), "logs deepseek-v4-flash-abc --all-containers=true --previous --tail 25") {
		t.Fatalf("unexpected pod log commands: %+v", podLogs)
	}
	if plan.PodLogSectionName(false) != "currentLogsByNamePrefix" || plan.PodLogSectionName(true) != "previousLogsByNamePrefix" {
		t.Fatalf("unexpected pod log section names")
	}
}

func assertPlanCommand(t *testing.T, commands []DiagnosticsCommand, name string, expected string) {
	t.Helper()
	for _, command := range commands {
		if command.Name == name && strings.Contains(strings.Join(command.Args, " "), expected) {
			return
		}
	}
	t.Fatalf("expected command %s containing %q, got %+v", name, expected, commands)
}
