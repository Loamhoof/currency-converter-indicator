package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	pb "github.com/loamhoof/indicator"
	"github.com/loamhoof/indicator/client"
)

var (
	icon, from, to, logFile string
	port                    int
	logger                  *log.Logger
)

func init() {
	flag.IntVar(&port, "port", 15000, "Port of the shepherd")
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

	id := fmt.Sprintf("indicator-currency-converter-%s-%s", from, to)

	sc := client.NewShepherdClient(port)
	for {
		err := sc.Init()
		if err == nil {
			break
		}
		logger.Fatalf("Could not connect: %v", err)

		time.Sleep(time.Second * 5)
	}
	defer sc.Close()

	iReq := &pb.Request{
		Id:         id,
		Icon:       icon,
		Label:      fmt.Sprintf("%s/%s: N/A", from, to),
		LabelGuide: "AAA/BBB: 123456789.123",
		Active:     true,
	}
	if _, err := sc.Update(iReq); err != nil {
		logger.Println(err)
	}

	for {
		iReq = &pb.Request{
			Id:         id,
			Icon:       icon,
			Label:      fmt.Sprintf("%s/%s: %.3f", from, to, get()),
			LabelGuide: "AAA/BBB: 123456789.123",
			Active:     true,
		}
		if _, err := sc.Update(iReq); err != nil {
			logger.Println(err)
		}

		time.Sleep(time.Minute)
	}
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
