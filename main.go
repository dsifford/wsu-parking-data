package main

import (
	"encoding/csv"
	"errors"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

type space struct {
	Name, Status, Updated, Available string
}

type structure struct {
	Name, URLCode string
	Number        int
	Spaces        []space
}

func main() {

	c := make(chan structure)
	go getStructures(c)
	for i := range c {
		writeData(i)
	}

}

func getStructures(c chan<- structure) {
	for i := 1; i < 9; i++ {
		if i == 7 || i == 4 {
			continue
		}

		s := structure{
			Name:    "Structure" + strconv.Itoa(i),
			Number:  i,
			URLCode: strconv.Itoa(i + 88),
		}

		// Attempt the request a total of 5 times
		var err error
		for j := 0; j < 5; j++ {
			s.Spaces, err = s.getSpaces()
			if err != nil {
				time.Sleep(time.Second * 5)
				continue
			}
			break
		}
		if err != nil {
			file, _ := os.OpenFile("/home/dsifford/Dropbox/ParkingData/errorlog.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
			defer file.Close()
			file.WriteString(err.Error())
		}

		c <- s
	}
	close(c)
}

func (s structure) getSpaces() ([]space, error) {

	spaces := []space{
		space{Name: "WSU Permit"},
		space{Name: "Student OneCard"},
		space{Name: "Visitor"},
	}
	re := map[string]*regexp.Regexp{
		"avail":   regexp.MustCompile(`([0-9]+|NONE)`),
		"status":  regexp.MustCompile(`(OPEN|CLOSED|FULL)`),
		"updated": regexp.MustCompile(`(?P<a>^.+: )(?P<b>.+)`),
	}

	// Request
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("GET", "http://m.wayne.edu/parking.php?location="+s.URLCode, nil)
	if err != nil {
		return spaces, errors.New("Request failed")
	}
	req.Header.Set("User-Agent", "Apple-iPhone6C1/")

	// Response
	resp, err := client.Do(req)
	if err != nil {
		return spaces, errors.New("Response failed")
	}
	defer resp.Body.Close()

	body, err := html.Parse(resp.Body)
	if err != nil {
		return spaces, errors.New("Error parsing response body")
	}

	// Parse relevant response data
	dataString, ok := scrape.Find(body, scrape.ByClass("available"))
	if !ok {
		return spaces, errors.New("Error: Line 105 - scrape.Find (available) -- not finding scrape info")
	}
	lastUpdated, ok := scrape.Find(body, scrape.ByClass("last_updated"))
	if !ok {
		return spaces, errors.New("Error: Line 109 - scrape.Find (last_updated) -- not finding scrape info")
	}

	avail := re["avail"].FindAllString(scrape.Text(dataString), -1)
	if len(avail) == 0 {
		avail = []string{"0", "0", "0"}
	}
	status := re["status"].FindAllString(scrape.Text(dataString), -1)
	if len(status) != 3 {
		return spaces, errors.New("Error: Line 118 - FindAllString (status) not returning 3 matches")
	}
	updated := re["updated"].FindStringSubmatch(scrape.Text(lastUpdated))
	if len(updated) == 0 {
		return spaces, errors.New("Error: Line 122 - FindAllStringSubmatch (updated) not finding a match")
	}

	for key := range spaces {
		spaces[key].Available = avail[key]
		spaces[key].Status = status[key]
		spaces[key].Updated = updated[2]
	}

	return spaces, nil

}

func writeData(s structure) {

	file, err := os.OpenFile("/home/dsifford/Dropbox/ParkingData/"+s.Name+".csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalln("CSV file could not be created or opened:", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	if stat, _ := file.Stat(); stat.Size() == 0 {
		writer.Write([]string{"Updated", "Type", "Status", "Spaces Available"})
	}

	for _, sp := range s.Spaces {
		writer.Write([]string{sp.Updated, sp.Name, sp.Status, sp.Available})
	}
	writer.Flush()

}
