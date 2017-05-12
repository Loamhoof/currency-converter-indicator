package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/conformal/gotk3/gtk"
	"github.com/doxxan/appindicator"
	"github.com/doxxan/appindicator/gtk-extensions/gotk3"
)

var (
	icon, from, to, logFile string
	indicator               *gotk3.AppIndicatorGotk3
	logger                  *log.Logger
)

func init() {
	flag.StringVar(&icon, "icon", "", "Path to the icon")
	flag.StringVar(&from, "from", "", "Base currency")
	flag.StringVar(&to, "to", "", "Target currency")
	flag.StringVar(&logFile, "log", "", "Log file")

	flag.Parse()

	if from == "" || to == "" {
		fmt.Println("Missing from/to currency.")
		os.Exit(1)
	}

	logger = log.New(os.Stdout, "", log.LstdFlags)
}

func main() {
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			logger.Fatalln(err)
		}
		defer f.Close()
		logger = log.New(f, "", log.LstdFlags)
	}

	gtk.Init(nil)

	id := fmt.Sprintf("indicator-currency-converter-%s-%s", from, to)
	indicator = gotk3.NewAppIndicator(id, icon, appindicator.CategorySystemServices)

	indicator.SetStatus(appindicator.StatusActive)
	indicator.SetLabel(fmt.Sprintf("%s/%s: N/A", from, to), "")

	menu, err := gtk.MenuNew()
	if err != nil {
		logger.Fatalln(err)
	}

	menuItem, err := gtk.MenuItemNewWithLabel("Refresh")
	if err != nil {
		logger.Fatalln(err)
	}

	menu.Append(menuItem)

	menuItem.Show()
	indicator.SetMenu(menu)

	menuItem.Connect("activate", refresh)

	go func() {
		ticker := time.Tick(time.Minute * 30)
		for {
			refresh()

			<-ticker
		}
	}()

	gtk.Main()
}

func refresh() {
	indicator.SetLabel(fmt.Sprintf("%s/%s: %.3f", from, to, get()), "")
}

func get() float64 {
	url := fmt.Sprintf("http://free.currencyconverterapi.com/api/v3/convert?q=%s_%s&compact=ultra", from, to)

	var (
		decoder *json.Decoder
		body    map[string]float64
		rate    float64
	)

	ticker := time.Tick(time.Second * 30)

Loop:
	logger.Println("Requesting", from, to)

	res, err := http.Get(url)
	if err != nil {
		logger.Println("Error", err)
		goto Wait
	}
	defer res.Body.Close()

	decoder = json.NewDecoder(res.Body)
	body = make(map[string]float64)

	if err := decoder.Decode(&body); err != nil {
		goto Wait
	}

	rate = body[fmt.Sprintf("%s_%s", from, to)]

	logger.Println("Got", rate)

	return rate

Wait:
	<-ticker

	goto Loop
}
