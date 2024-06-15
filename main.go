package main

import (
	pb "collegeSync/proto"
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"google.golang.org/grpc"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func getTime(page string) (string, error) {
	upperIndex := strings.Index(page, "<table>")
	if upperIndex == 1 {
		return "", errors.New("time not found")
	}
	page = page[upperIndex:]
	lowerIndex := strings.Index(page, "</table>")
	if lowerIndex == -1 {
		return "", errors.New("time not found")
	}
	page = page[:lowerIndex+len("</table>")]
	return page, nil
}

func createJson(timeHtml string, groupHtml string) {
	type Schedule struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Class string `json:"class"`
	}
	fmt.Println(groupHtml)
	var schedule []Schedule
	dateFormat := func(str string) string {
		date, _ := time.Parse("02.01.2006", str)
		return date.Format("2006-01-02")
	}
	timeFormat := func(str string, x bool) string {
		if x {
			return strings.Split(str, "-")[0] + ":00"
		} else {
			return strings.Split(str, "-")[1] + ":00"
		}
	}
	var DateArray []string
	var WeekdayTime []string
	var SaturdayTime []string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(groupHtml))
	if err != nil {
		log.Fatalf("Error loading html document. %s", err)
	}
	doc.Find("tr").First().Find("td").Each(func(i int, s *goquery.Selection) {
		if i > 1 {
			ind := strings.Index(s.Text(), ", ")
			DateArray = append(DateArray, dateFormat(s.Text()[ind+len(", "):])+"T")
		}
	})
	timeDoc, err := goquery.NewDocumentFromReader(strings.NewReader(timeHtml))
	if err != nil {
		log.Fatalf("Error loading html document. %s", err)
	}
	timeDoc.Find("tr").Each(func(i int, s *goquery.Selection) {
		if i >= 1 {
			SaturdayTime = append(SaturdayTime, s.Find("td").Last().Text())
		}
		if i != 0 {
			tds := s.Find("td")
			WeekdayTime = append(WeekdayTime, tds.Eq(tds.Length()-2).Text())
		}
	})

	doc.Find("tr").Each(func(i int, tr *goquery.Selection) {
		if i >= 2 {
			tr.Find("td").Each(func(j int, td *goquery.Selection) {
				if j >= 2 && j-2 < len(WeekdayTime) {
					schedule = append(schedule, Schedule{Start: DateArray[j-2] + timeFormat(WeekdayTime[i-2], true), End: DateArray[j-2] + timeFormat(WeekdayTime[i-2], false), Class: td.Text()})
				}
				if j == 7 {
					schedule = append(schedule, Schedule{Start: DateArray[j-2] + timeFormat(SaturdayTime[i-2], true), End: DateArray[j-2] + timeFormat(SaturdayTime[i-2], false), Class: td.Text()})
				}
			})
		}
	})
	scheduleRequest := &pb.ScheduleRequest{}
	for _, s := range schedule {
		entry := &pb.ScheduleEntry{
			Start:     s.Start,
			End:       s.End,
			ClassName: s.Class,
		}
		scheduleRequest.Entries = append(scheduleRequest.Entries, entry)
	}
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewParserClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := client.SendSchedule(ctx, scheduleRequest)
	if err != nil {
		log.Fatalf("could not send request: %v", err)
	}
	log.Printf(response.Status)
}

func extractGroup(page string, groupName string) (string, error) {
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
	groupHtml, err := extractGroup(string(body), "ИСП-911")
	if err != nil {
		log.Fatal(err)
	}
	timeHtml, err := getTime(string(body))
	if err != nil {
		log.Fatal(err)
	}
	createJson(timeHtml, groupHtml)
}
