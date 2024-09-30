package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type Resource struct {
	resourceName string
	path         string
	env          map[string]string
}

func CreateDockerNetwork() error {
	var stderr bytes.Buffer
	cmd := exec.Command("docker", "network", "create", "otel-net")
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v", stderr.String())
	}

	return nil
}

func RemoveDockerNetwork() error {
	cmd := exec.Command("docker", "network", "rm", "otel-net")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func UpDockerCompose(resource Resource) error {
	cmd := exec.Command("docker", "compose", "-f", "docker-compose.yml", "up", "-d")
	cmd.Dir = resource.path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for k, v := range resource.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	return cmd.Run()
}

func DownDockerCompose(resource Resource) error {
	cmd := exec.Command("docker", "compose", "-f", "docker-compose.yml", "down")
	cmd.Dir = resource.path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func RunJarWithJavaAgent() error {
	jarName := "spring-petclinic-3.3.0-SNAPSHOT.jar"
	resource := Resource{
		resourceName: "java-application",
		path:         "./java-application",
		env: map[string]string{
			"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:9987",
			"OTEL_EXPORTER_OTLP_PROTOCOL": "grpc",
			"OTEL_TRACES_EXPORTER":        "otlp",
			"OTEL_METRICS_EXPORTER":       "otlp",
			"OTEL_LOGS_EXPORTER":          "otlp",
		},
	}

	cmd := exec.Command("javaw", "-javaagent:opentelemetry-javaagent.jar", "-jar", jarName)
	cmd.Dir = resource.path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for k, v := range resource.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	fmt.Printf("%s is Run", resource.resourceName)

	return cmd.Start()
}

func KillJar() error {
	cmd := exec.Command("cmd.exe", "/C", "netstat -ano | findstr 8080")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	// 출력 결과를 문자열로 변환합니다.
	outputStr := string(output)

	// 출력 결과에서 PID를 추출합니다.
	lines := strings.Split(outputStr, "\n")
	trim := strings.TrimSpace(lines[0])
	fields := strings.Fields(trim)

	pid := fields[len(fields)-1]

	killCmd := exec.Command("cmd.exe", "/C", fmt.Sprintf("taskkill /f /pid %s", pid))
	return killCmd.Run()
}

func DockerLogs(svc string) error {
	cmd := exec.Command("docker", "ps")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	// 출력된 결과를 줄 단위로 분리
	lines := strings.Split(string(output), "\n")
	var pid string

	// 각 줄을 검사하여 서비스 이름이 포함된 줄만 출력
	for _, line := range lines {
		if strings.Contains(line, svc) {
			pid = strings.Fields(line)[0]
		}
	}

	if pid == "" {
		return errors.New("pid not found")
	}

	fmt.Println("pid:", pid)

	logsCmd := exec.Command("docker", "logs", "-f", pid)

	logsCmd.Stdout = os.Stdout
	logsCmd.Stderr = os.Stderr
	return logsCmd.Run()
}

func main() {
	up := flag.Bool("up", false, "Bring up the docker compose services")
	down := flag.String("down", "", "Bring down the docker compose services")
	jar := flag.Bool("jar", false, "Run the jar with java agent")
	kill := flag.Bool("kill", false, "Run the kill java application")
	logs := flag.String("logs", "", "logging specific service")

	flag.Parse()

	resources := []Resource{
		{"otel-collector", "./otel-collector", nil},
		{"opensearch", "./opensearch", map[string]string{"OPENSEARCH_INITIAL_ADMIN_PASSWORD": "QWERqwer1!"}},
		{"data-prepper", "./data-prepper", nil},
		{"prometheus", "./prometheus", nil},
	}

	var action string
	switch {
	case *up:
		action = "up"
	case *down != "":
		action = "down"
	case *jar:
		action = "jar"
	case *kill:
		action = "kill"
	case *logs != "":
		action = "logs"
	default:
		action = "usage"
	}

	switch action {
	case "up":
		if err := CreateDockerNetwork(); err != nil {
			if strings.Contains(err.Error(), "network with name otel-net already exists") {
				log.Println("Network already exists, continuing...")
			} else {
				log.Fatalf("Failed to create docker network: %v", err)
			}
		}

		for _, resource := range resources {
			if err := UpDockerCompose(resource); err != nil {
				log.Printf("Error bringing up resource %s: %v", resource.resourceName, err)
				for _, res := range resources {
					DownDockerCompose(res)
				}
				RemoveDockerNetwork()
				log.Fatalf("docker compose up error: %v", err)
			}
		}

	case "down":
		if *down == "all" {
			for _, resource := range resources {
				if err := DownDockerCompose(resource); err != nil {
					log.Printf("Error bringing down resource %s: %v", resource.resourceName, err)
				}
			}
			RemoveDockerNetwork()
		} else {
			for _, resource := range resources {
				if strings.Contains(*down, resource.resourceName) {
					if err := DownDockerCompose(resource); err != nil {
						log.Printf("Error bringing down resource %s: %v", resource.resourceName, err)
					}
				}
			}
		}

	case "jar":
		if err := RunJarWithJavaAgent(); err != nil {
			log.Fatalf("Failed to run jar with java agent: %v", err)
		}

	case "kill":
		if err := KillJar(); err != nil {
			log.Fatalf("Failed to kill: %v", err)
		} else {
			fmt.Println("java application killed")
		}
	case "logs":
		if err := DockerLogs(*logs); err != nil {
			log.Fatalf("Failed to logging: %v", err)
		}

	case "usage":
		flag.Usage()
	}
}
