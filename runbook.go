package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"syscall"
	"bufio"
)

// runBook represents a collection of scripts.
type runBook struct {
	Scripts         []script `json:"scripts"`
	AllowedNetworks Networks `json:"allowedNetworks,omitempty"`
}

type runBookResponse struct {
	Results []result `json:"results"`
}

type result struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	StatusCode int    `json:"status_code"`
}

type script struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// Networks is its own struct for JSON unmarshalling gymnastics
type Networks struct {
	Networks []net.IPNet
}

// UnmarshalJSON for custom type Networks
func (nets *Networks) UnmarshalJSON(data []byte) error {
	ns := []string{}
	if err := json.Unmarshal(data, &ns); err != nil {
		return err
	}

	nets.Networks = make([]net.IPNet, len(ns))
	for i, nw := range ns {
		_, ipnet, err := net.ParseCIDR(nw)
		if err != nil {
			return err
		}
		nets.Networks[i] = *ipnet
	}
	return nil
}

// NewRunBook returns the runBook identified by id.
func NewRunBook(id string) (*runBook, error) {
	return getRunBookById(id)
}

func (r *runBook) AddrIsAllowed(remoteIP net.IP) bool {
	if len(r.AllowedNetworks.Networks) == 0 {
		return true
	}
	for _, nw := range r.AllowedNetworks.Networks {
		if nw.Contains(remoteIP) {
			return true
		}
	}
	return false
}

func (r *runBook) execute() (*runBookResponse, error) {
	results := make([]result, 0)
	for _, x := range r.Scripts {
		r, err := execScript(x)
		if err != nil {
			log.Println("ERROR :" + err.Error())
		}
		results = append(results, r)
	}
	return &runBookResponse{results}, nil
}

func execScript(s script) (result, error) {
	cmd := exec.Command(s.Command, s.Args...)

	// Get the stdout and stderr pipes for real-time streaming
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return result{}, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return result{}, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command execution
	err = cmd.Start()
	if err != nil {
		return result{}, fmt.Errorf("failed to start script: %w", err)
	}

	// Create goroutines to stream stdout and stderr to Go's standard output in real-time
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			// Log each line of stdout as it is produced
			log.Printf("stdout: %s", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Error reading stdout: %v", err)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			// Log each line of stderr as it is produced
			log.Printf("stderr: %s", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Error reading stderr: %v", err)
		}
	}()

	// Wait for the command to complete
	err = cmd.Wait()

	// Create the result object
	r := result{
		Stdout: "", // Stdout and stderr are already being printed in real-time
		Stderr: "",
		StatusCode: -1,
	}

	if err == nil {
		r.StatusCode = cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
	} else {
		// If there's an error, log the error message
		log.Printf("Error executing script %s: %s", s.Command, err.Error())
	}

	return r, err
}


func getRunBookById(id string) (*runBook, error) {
	var r = new(runBook)
	runBookPath := fmt.Sprintf("%s/%s.json", configdir, id)
	data, err := ioutil.ReadFile(runBookPath)
	if err != nil {
		return r, fmt.Errorf("cannot read run book %s: %s", runBookPath, err)
	}
	err = json.Unmarshal(data, r)
	if err != nil {
		return r, err
	}
	return r, nil
}
