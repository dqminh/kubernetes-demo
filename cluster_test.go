// the cluster consists of 4 nodes
// - master is the scheduler/apiserver/etcd server
// - balancer-01 proxy traffic to pods
// - node-01 can schedule pods
// - node-02 can schedule pods
//
// This assumes that we already spin up the cluster. Its default setings are:
// - 1 redis container
// - 3 go containers
// - 3 nginx containers
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tsenart/vegeta/lib"
)

var (
	Endpoint = "http://172.17.8.102/"
	Rate     = uint64(50)
	Duration = 5 * time.Second
)

func init() {
	durationEnv := os.Getenv("DURATION")
	d, err := strconv.ParseInt(durationEnv, 10, 64)
	if err == nil {
		Duration = time.Duration(d) * time.Second
	}

	rateEnv := os.Getenv("RATE")
	r, err := strconv.ParseUint(rateEnv, 10, 64)
	if err == nil {
		Rate = r
	}
}

func TestLoadNormal(t *testing.T) {
	metrics := hit(Endpoint, Rate, Duration, make(chan struct{}))
	printMetrics(t, metrics)
}

func TestShutdown1Nginx(t *testing.T) {
	resCh := make(chan *vegeta.Metrics)
	cancel := make(chan struct{})
	go func() {
		resCh <- hit(Endpoint, Rate, Duration, cancel)
	}()
	time.Sleep(1 * time.Second)

	pod, err := podsAt("name=nginx", 0)
	if err != nil {
		close(cancel)
		t.Fatalf("failed to get pod %s", "name=nginx")
	}

	output, err := exec.Command("kubectl", "stop", "pods", pod).CombinedOutput()
	if err != nil {
		close(cancel)
		t.Fatalf("Output(%s), err %v", "name=nginx", string(output), err)
	}
	t.Logf("stop pod: %s", string(output))
	printMetrics(t, <-resCh)
}

func TestShutdownAllNginx(t *testing.T) {
	resCh := make(chan *vegeta.Metrics)
	cancel := make(chan struct{})
	go func() {
		resCh <- hit(Endpoint, Rate, Duration, cancel)
	}()
	time.Sleep(1 * time.Second)

	output, err := exec.Command("kubectl", "stop", "pods", "-l", "name=nginx").CombinedOutput()
	if err != nil {
		close(cancel)
		t.Fatalf("Output(%s), err %v", "name=nginx", string(output), err)
	}
	t.Logf("stop pods: %s", string(output))
	printMetrics(t, <-resCh)
}

func TestShutdown1Go(t *testing.T) {
	resCh := make(chan *vegeta.Metrics)
	cancel := make(chan struct{})
	go func() {
		resCh <- hit(Endpoint, Rate, Duration, cancel)
	}()
	time.Sleep(1 * time.Second)

	label := "name=guestbook"
	pod, err := podsAt(label, 0)
	if err != nil {
		close(cancel)
		t.Fatalf("failed to get pod %s", label)
	}

	output, err := exec.Command("kubectl", "stop", "pods", pod).CombinedOutput()
	if err != nil {
		close(cancel)
		t.Fatalf("Output(%s), err %v", label, string(output), err)
	}
	t.Logf("stop pod: %s", string(output))
	printMetrics(t, <-resCh)
}

func TestShutdownAllGo(t *testing.T) {
	resCh := make(chan *vegeta.Metrics)
	cancel := make(chan struct{})
	go func() {
		resCh <- hit(Endpoint, Rate, Duration, cancel)
	}()
	time.Sleep(1 * time.Second)

	// random between 3 guestbook go app!
	label := "name=guestbook"
	output, err := exec.Command("kubectl", "stop", "pods", "-l", label).CombinedOutput()
	if err != nil {
		close(cancel)
		t.Fatalf("Output(%s), err %v", label, string(output), err)
	}
	t.Logf("stop pod: %s", string(output))
	printMetrics(t, <-resCh)
}

func TestLoadNormalFinal(t *testing.T) {
	metrics := hit(Endpoint, Rate, Duration, make(chan struct{}))
	printMetrics(t, metrics)
}

// hit runs a load test on the specified endpoints with the given rate and
// duration.
func hit(endpoint string, rate uint64, duration time.Duration, cancel <-chan struct{}) *vegeta.Metrics {
	targeter := vegeta.NewStaticTargeter(&vegeta.Target{
		Method: "GET",
		URL:    endpoint,
	})
	attacker := vegeta.NewAttacker()

	var results vegeta.Results
	ch := attacker.Attack(targeter, rate, duration)
	var done bool
	for !done {
		select {
		case res, ok := <-ch:
			if ok {
				results = append(results, res)
			} else {
				done = true
			}
		case <-cancel:
			attacker.Stop()
			done = true
		}
	}
	return vegeta.NewMetrics(results)
}

// get all pods' name with the given label
func pods(label string) (names []string, err error) {
	output, err := exec.Command("kubectl", "get", "pods", "-l", label).Output()
	if err != nil {
		return names, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(strings.TrimSpace(line))
		if fields[0] != "POD" {
			names = append(names, fields[0])
		}
	}
	return names, scanner.Err()
}

// get the name of the pods with specific label at the given index.
func podsAt(label string, index int) (string, error) {
	names, err := pods(label)
	if err != nil {
		return "", err
	}
	if len(names)-1 < index {
		return "", fmt.Errorf("index too large %d, has %d", index, len(names))
	}
	return names[index], nil
}

func printMetrics(t *testing.T, metrics *vegeta.Metrics) {
	t.Logf("p50 %v", metrics.Latencies.P50)
	t.Logf("p95  %v", metrics.Latencies.P95)
	t.Logf("p99 %v", metrics.Latencies.P99)
	t.Logf("mean %v", metrics.Latencies.Mean)
	t.Logf("wait %s", metrics.Wait)
	t.Logf("success %.2f", metrics.Success*100)
	t.Logf("requests %d", metrics.Requests)
	if metrics.Success < 1 {
		t.Logf("errors %v", metrics.Errors)
	}
}
