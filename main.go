package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/conformal/gotk3/gtk"
	"github.com/doxxan/appindicator"
	"github.com/doxxan/appindicator/gtk-extensions/gotk3"
)

var (
	icon, from, to string
	indicator      *gotk3.AppIndicatorGotk3
)

func init() {
	flag.StringVar(&icon, "icon", "", "Path to the icon")
	flag.StringVar(&from, "from", "", "Base currency")
	flag.StringVar(&to, "to", "", "Target currency")

	flag.Parse()

	if from == "" || to == "" {
		fmt.Println("Missing from/to currency.")
		os.Exit(1)
	}
}

func main() {
	gtk.Init(nil)

	id := fmt.Sprintf("currency-indicator-%s-%s", from, to)
	indicator = gotk3.NewAppIndicator(id, icon, appindicator.CategorySystemServices)

	indicator.SetStatus(appindicator.StatusActive)
	indicator.SetLabel(fmt.Sprintf("%s/%s: N/A", from, to), "")

	menu, err := gtk.MenuNew()
	if err != nil {
		panic(err)
	}
	indicator.SetMenu(menu)

	go refresh()

	gtk.Main()
}

func refresh() {
	ticker := time.Tick(time.Minute * 30)
	for {
		v, err := get()
		if err != nil {
			panic(err)
		}

		indicator.SetLabel(fmt.Sprintf("%s/%s: %.3f", from, to, v), "")

		<-ticker
	}
}

func get() (float64, error) {
	url := fmt.Sprintf("http://free.currencyconverterapi.com/api/v3/convert?q=%s_%s&compact=ultra", from, to)

	res, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	body := make(map[string]float64)

	if err := decoder.Decode(&body); err != nil {
		return 0, err
	}

	return body[fmt.Sprintf("%s_%s", from, to)], nil
}
