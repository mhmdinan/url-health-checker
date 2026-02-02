package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type Target struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	CheckType string `json:"check_type"`
}

type TargetInFile []Target

const fileName = "targets.json"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: check | add <name> <address> <type>")
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "add":
		if len(os.Args) < 5 {
			fmt.Println("Missing arguments. Usage: add <name> <address> <type>")
			return
		}
		newTarget := Target{
			Name:      os.Args[2],
			Address:   os.Args[3],
			CheckType: os.Args[4],
		}
		addTarget(newTarget)
	case "check":
		checkTargets()
	default:
		fmt.Println("Unknown command")
	}
}

func addTarget(newTarget Target) {
	targets, err := loadTargets()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Adding first target")
			saveTargets([]Target{newTarget})
			return
		}
		fmt.Println("Error:", err)
		return
	}
	targets = append(targets, newTarget)
	saveTargets(targets)
	fmt.Printf("Added %s (%s) to targets to check.\n", newTarget.Name, newTarget.CheckType)
}

func checkTargets() {
	targets, err := loadTargets()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	var wg sync.WaitGroup

	fmt.Printf("Checking %d Endpoints...\n\n", len(targets))

	for _, t := range targets {
		wg.Add(1)
		go func(target Target) {
			defer wg.Done()

			switch target.CheckType {
			case "http":
				runHTTPCheck(target)
			case "ping":
				runPingCheck(target)
			default:
				fmt.Printf("Unknown check type for %s\n", target.Name)
			}
		}(t)
	}
	wg.Wait()
}

func runHTTPCheck(t Target) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(t.Address)
	if err != nil {
		fmt.Printf(" [HTTP] %s: Down (%v) \n", t.Name, err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("[HTTP] %s: Status %d \n", t.Name, resp.StatusCode)
}

func runPingCheck(t Target) {
	timeout := 5 * time.Second

	start := time.Now()

	conn, err := net.DialTimeout("tcp", t.Address, timeout)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("[PING] %s (%s): Down (%v) \n", t.Name, t.Address, err)
		return
	}
	conn.Close()
	fmt.Printf("[PING] %s (%s): Connected (%v) \n", t.Name, t.Address, duration)
}

func loadTargets() (TargetInFile, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("no targets found. Please add target to file")
			return nil, err
		}
		fmt.Printf("Error occured: %v\n", err)
		return nil, err
	}
	var targets []Target
	if err := json.Unmarshal(data, &targets); err != nil {
		fmt.Printf("Error occured in parsing json file: %v\n", err)
		return nil, err
	}
	return targets, nil

}

func saveTargets(targets []Target) {
	data, err := json.MarshalIndent(targets, "", " ")
	if err != nil {
		fmt.Printf("Error encoding Json: %v\n", err)
		return
	}

	err = os.WriteFile(fileName, data, 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
	}
}
