package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func extractGroup(page string, groupName string) (string, error) { //TODO: Think about passing the page by reference
	upperIndex := strings.Index(page, fmt.Sprintf(`"%s":[%s`, groupName, "`"))
	if upperIndex == -1 {
		return "", errors.New(fmt.Sprintf("group %s not found", groupName))
	}
	page = page[upperIndex+len(fmt.Sprintf(`"%s":[%s`, groupName, "`")):]
	lowerIndex := strings.Index(page, fmt.Sprintf(`%s,],`, "`"))
	if lowerIndex == -1 {
		return "", errors.New(fmt.Sprintf("group %s not found", groupName))
	}
	page = "<table>" + strings.ReplaceAll(strings.ReplaceAll(page[:lowerIndex], "`, `", ""), "`,`", "") + "</table>"
	return page, nil
}

func main() {
	resp, err := http.Get("https://ppk.sstu.ru/schedule")
	if err != nil {
		log.Fatalf("Error sending request %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request completed with code %s", resp.Status)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println(err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response %s", err)
	}
	groupHtml, err := extractGroup(string(body), "ИСП-934")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(groupHtml)
}
