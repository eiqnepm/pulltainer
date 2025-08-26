package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var client = &http.Client{Timeout: 10 * time.Second}

type Stack struct {
	ID         int    `json:"Id"`
	Name       string `json:"Name"`
	EndpointID int    `json:"EndpointId"`
	Env        []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"Env"`
	Status  int    `json:"Status"`
	Webhook string `json:"Webhook"`
}

var (
	cronSpec        = os.Getenv("PULLTAINER_CRON")
	portainerURL    = os.Getenv("PULLTAINER_URL")
	portainerAPIKey = os.Getenv("PULLTAINER_API_KEY")
)

func getStacks() ([]Stack, error) {
	u, err := url.JoinPath(portainerURL, "api", "stacks")
	if err != nil {
		return nil, fmt.Errorf("join path: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("X-API-Key", portainerAPIKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("error: get stacks: close: ", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}

	data := []Stack{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return data, nil
}

func getStackImagesStatus(id int) (string, error) {
	u, err := url.JoinPath(portainerURL, "api", "stacks", strconv.Itoa(id), "images_status")
	if err != nil {
		return "", fmt.Errorf("join path: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("X-API-Key", portainerAPIKey)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("do: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("error: get stack images status: close: ", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read all: %w", err)
	}

	data := struct {
		Message string
		Status  string
	}{}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}

	return data.Status, nil
}

func getStackFile(id int) (string, error) {
	u, err := url.JoinPath(portainerURL, "api", "stacks", strconv.Itoa(id), "file")
	if err != nil {
		return "", fmt.Errorf("join path: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("X-API-Key", portainerAPIKey)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("do: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("error: get stack file: close: ", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read all: %w", err)
	}

	data := struct {
		StackFileContent string
	}{}

	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}

	return data.StackFileContent, nil
}

func putStack(stack Stack, stackFile string) error {
	data := struct {
		ID               int    `json:"id"`
		StackFileContent string `json:"StackFileContent"`
		Env              []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"Env"`
		Prune     bool   `json:"Prune"`
		Webhook   string `json:"Webhook"`
		PullImage bool   `json:"PullImage"`
	}{
		ID:               stack.ID,
		StackFileContent: stackFile,
		Env:              stack.Env,
		Prune:            true,
		Webhook:          stack.Webhook,
		PullImage:        true,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("body: %w", err)
	}

	u, err := url.Parse(portainerURL)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	u.Path = path.Join(u.Path, "api", "stacks", strconv.Itoa(stack.ID))
	q := u.Query()
	q.Set("endpointId", strconv.Itoa(stack.EndpointID))
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodPut, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("X-API-Key", portainerAPIKey)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("status: %w", errors.New(resp.Status))
	}

	return nil
}

func main() {
	c := cron.New()
	if cronSpec == "" {
		cronSpec = "0 4 * * *"
	}

	beAPI := strings.EqualFold(os.Getenv("PULLTAINER_BE_API"), "true")
	cmd := func() {
		stacks, err := getStacks()
		if err != nil {
			log.Println(fmt.Errorf("error: get stacks: %w", err))
			return
		}

		for _, stack := range stacks {
			// Stack is not running
			if stack.Status != 1 {
				log.Printf("skipping as stack is not running: %v\n", stack.Name)
				continue
			}

			ignore := false
			for _, env := range stack.Env {
				if strings.EqualFold(env.Name, "PULLTAINER_IGNORE") {
					ignore = true
					break
				}
			}

			if ignore {
				log.Printf("skipping as stack is ignored: %v\n", stack.Name)
				continue
			}

			if beAPI {
				stackImagesStatus, err := getStackImagesStatus(stack.ID)
				if err != nil {
					log.Println(fmt.Errorf("error: stack images status: %w", err))
					continue
				}

				if stackImagesStatus != "outdated" {
					log.Printf("skipping as stack is not outdated: %v\n", stack.Name)
					continue
				}
			}

			stackFile, err := getStackFile(stack.ID)
			if err != nil {
				log.Println(fmt.Errorf("error: stack file: %w", err))
				continue
			}

			if err := putStack(stack, stackFile); err != nil {
				log.Println(fmt.Errorf("error: put stack: %w", err))
				continue
			}

			log.Printf("stack redeployed: %v\n", stack.Name)
		}

		log.Printf("next job: %v\n", c.Entries()[0].Schedule.Next(time.Now()).String())
	}

	c.AddFunc(cronSpec, cmd)
	cmd()
	c.Run()
}
