package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

var (
	from, to string
	port     int
	logger   *log.Logger

	symbols = map[string]string{
		"EUR": "€",
		"JPY": "¥",
	}
)

func init() {
	flag.IntVar(&port, "port", 15000, "Port of the shepherd")
	flag.StringVar(&from, "from", "", "Base currency")
	flag.StringVar(&to, "to", "", "Target currency")

	flag.Parse()

	if from == "" || to == "" {
		fmt.Println("Missing from/to currency.")
		os.Exit(1)
	}

	logger = log.New(os.Stdout, "", log.LstdFlags)
}

func main() {
	id := fmt.Sprintf("indicator-currency-converter-%s-%s", from, to)

	label := fmt.Sprintf("%s/%s: N/A", symbol(from), symbol(to))
	if err := update(id, label); err != nil {
		logger.Println(err)
	}

	feedC := make(chan float64)
	doneC := make(chan struct{})

	go feed(feedC, doneC)

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt)

	for {
		select {
		case v := <-feedC:
			label := fmt.Sprintf("%s/%s: %.2f", symbol(from), symbol(to), v)
			if err := update(id, label); err != nil {
				logger.Println(err)
			}
		case <-sigC:
			close(doneC)
			os.Exit(0)
		}
	}
}

func feed(feedC chan<- float64, doneC <-chan struct{}) {
	url := fmt.Sprintf("http://free.currencyconverterapi.com/api/v3/convert?q=%s_%s&compact=ultra", from, to)

	rate, err := get(url)
	if err != nil {
		logger.Println(err)
	}

	feedC <- rate

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rate, err := get(url)
			if err != nil {
				logger.Println(err)
				break
			}

			feedC <- rate
		case <-doneC:
			return
		}
	}
}

func get(url string) (float64, error) {
	logger.Println("Requesting", from, to)

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

	rate := body[fmt.Sprintf("%s_%s", from, to)]

	logger.Println("Got", rate)

	return rate, nil
}

func update(id, label string) error {
	resp, err := http.Post(fmt.Sprintf("http://localhost:%v/%s", port, id), "text/plain", strings.NewReader(label))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func symbol(currency string) string {
	if s, ok := symbols[currency]; ok {
		return s
	}

	return currency
}
