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
				continue
			}
			break
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
		"avail":   regexp.MustCompile(`[0-9]+`),
		"status":  regexp.MustCompile(`(OPEN|CLOSED|FULL)`),
		"updated": regexp.MustCompile(`(?P<1>^.+: )(?P<2>.+)`),
	}

	// Request
	client := &http.Client{}
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
	dataString, _ := scrape.Find(body, scrape.ByClass("available"))
	lastUpdated, _ := scrape.Find(body, scrape.ByClass("last_updated"))

	avail := re["avail"].FindAllString(scrape.Text(dataString), -1)
	if len(avail) == 0 {
		avail = []string{"0", "0", "0"}
	}
	status := re["status"].FindAllString(scrape.Text(dataString), -1)
	updated := re["updated"].FindStringSubmatch(scrape.Text(lastUpdated))[2]

	for key := range spaces {
		spaces[key].Available = avail[key]
		spaces[key].Status = status[key]
		spaces[key].Updated = updated
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
