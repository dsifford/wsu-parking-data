package main

import (
	"fmt"
	"github.com/tealeg/xlsx"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

type space struct {
	Name, Status, Updated string
	Available             int
}

type structure []space

func main() {
	data := getData()
	file := openFile()
	saveData(file, data)
}

func getData() structure {

	structure6 := structure{
		space{Name: "WSU Permit"},
		space{Name: "Student OneCard"},
		space{Name: "Visitor"},
	}
	re := map[string]*regexp.Regexp{
		"avail":   regexp.MustCompile(`[0-9]+`),
		"status":  regexp.MustCompile(`(OPEN|CLOSED)`),
		"updated": regexp.MustCompile(`(?P<1>^.+: )(?P<2>.+)`),
	}

	// TODO: Implement this for all structures.
	// Parking structure URL "location"s are numbered starting from 89
	// Thus...
	// 1 = 89, 2 = 90, 3 = 91, 4 = 92, 5 = 93, 6 = 94, 8 = 96

	// Request
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://m.wayne.edu/parking.php?location=94", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", "Apple-iPhone6C1/")

	// Response
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	// Parse relevant response data
	dataString, _ := scrape.Find(body, scrape.ByClass("available"))
	lastUpdated, _ := scrape.Find(body, scrape.ByClass("last_updated"))

	avail := re["avail"].FindAllString(scrape.Text(dataString), -1)
	status := re["status"].FindAllString(scrape.Text(dataString), -1)
	updated := re["updated"].FindStringSubmatch(scrape.Text(lastUpdated))[2]

	fmt.Println(updated)

	for key := range structure6 {
		structure6[key].Available, _ = strconv.Atoi(avail[key])
		structure6[key].Status = status[key]
		structure6[key].Updated = updated
	}

	return structure6

}

func openFile() (file *xlsx.File) {

	// TODO: Pick an actual directory for this to run in (it's going to run as a cron task)

	if _, err := os.Stat("/home/dsifford/Dropbox/ParkingData/data.xlsx"); os.IsNotExist(err) {

		file := xlsx.NewFile()
		sheet, err := file.AddSheet("Structure 6")
		if err != nil {
			panic(err)
		}

		headingStyle := xlsx.NewStyle()
		headingStyle.Font.Bold = true

		row := sheet.AddRow()
		for _, name := range []string{"Updated", "Type", "Status", "Spaces Available"} {
			cell := row.AddCell()
			cell.SetStyle(headingStyle)
			cell.SetString(name)
		}

		return file

	}

	file, err := xlsx.OpenFile("/home/dsifford/Dropbox/ParkingData/data.xlsx")
	if err != nil {
		panic(err)
	}
	return file

}

func saveData(f *xlsx.File, s structure) {

	// FIXME: Fix dirty code below
	for _, sheet := range f.Sheets {
		for _, spaceType := range s {

			row := sheet.AddRow()

			cell := row.AddCell()
			cell.SetString(spaceType.Updated)

			cell = row.AddCell()
			cell.SetString(spaceType.Name)

			cell = row.AddCell()
			cell.SetString(spaceType.Status)

			cell = row.AddCell()
			cell.SetInt(spaceType.Available)
		}
	}

	if err := f.Save("/home/dsifford/Dropbox/ParkingData/data.xlsx"); err != nil {
		panic(err)
	}

}
