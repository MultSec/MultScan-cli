package main

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"

	"github.com/mgutz/ansi"
)

// Define a struct for the known parts of the JSON structure
type Machine struct {
    MachineIP   string `json:"machine_ip"`
    MachineName string `json:"machine_name"`
}

type Log int64

const (
    logError Log = iota
    logInfo
    logStatus
    logInput
	logSuccess
	logSection
	logSubSection
)

// Function to print logs
func printLog(log Log, text string) {
	switch log {
	case logError:
		fmt.Printf("[%s] %s %s\n", ansi.ColorFunc("red")("!"), ansi.ColorFunc("red")("ERROR:"), ansi.ColorFunc("cyan")(text))
	case logInfo:
		fmt.Printf("[%s] %s\n", ansi.ColorFunc("blue")("i"), text)
	case logStatus:
		fmt.Printf("[*] %s\n", text)
	case logInput:
		fmt.Printf("[%s] %s", ansi.ColorFunc("yellow")("?"), text)
	case logSuccess:
		fmt.Printf("[%s] %s\n", ansi.ColorFunc("green")("+"), text)
	case logSection:
		fmt.Printf("\t[%s] %s\n", ansi.ColorFunc("yellow")("-"), text)
	case logSubSection:
		fmt.Printf("\t\t[%s] %s\n", ansi.ColorFunc("magenta")(">"), text)
	}
}

// Function to get machines from the server
func getMachines(ip string, port int) ([]Machine, error) {
	var machines []Machine

	printLog(logInfo, fmt.Sprintf("%s %s", ansi.ColorFunc("default+hb")("Server IP: "), ansi.ColorFunc("cyan")(ip)))
	printLog(logInfo, fmt.Sprintf("%s %s", ansi.ColorFunc("default+hb")("Server Port: "), ansi.ColorFunc("cyan")(fmt.Sprintf("%d", port))))

	url := fmt.Sprintf("http://%s:%d/api/v1/machines", ip, port)
	resp, err := http.Get(url)
	if err != nil {
		return machines, fmt.Errorf("failed to fetch machines: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return machines, fmt.Errorf("server returned non-200 status: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return machines, fmt.Errorf("failed to read response body: %v", err)
	}

	if err := json.Unmarshal(body, &machines); err != nil {
		return machines, fmt.Errorf("failed to parse known parts of JSON: %v", err)
	}

	return machines, nil
}

// Function to display machines in a readable format
func displayMachines(machines []Machine) {
	printLog(logInfo, "Retrieved machines from server")
	for _, machine := range machines {
		printLog(logSection, fmt.Sprintf("%s:", ansi.ColorFunc("default+hb")(machine.MachineName)))
		printLog(logSubSection, fmt.Sprintf("%s", ansi.ColorFunc("cyan")(machine.MachineIP)))
    }
}