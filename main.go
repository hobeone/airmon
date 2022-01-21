package main

import (
	"expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hobeone/airmon/sds011"
)

var (
	portPath   = flag.String("port_path", "/dev/ttyUSB0", "serial port path")
	listenPort = flag.Int("listen_port", 8080, "port to listen and publish counts")
	debugFlag  = flag.Bool("debug", false, "Output debug info")
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr,
			`sds011 reads data from the SDS011 sensor and exports them on a HTTP interface at /debug/vars

If the debug flag is given it will also output to the console with the following format:
an RFC3339 timestamp, the PM2.5 level, the PM10 level`)
		fmt.Fprintf(os.Stderr, "\n\nUsage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	url := fmt.Sprintf("localhost:%d", *listenPort)
	fmt.Printf("Listening on %s\n", url)
	go http.ListenAndServe(url, nil)

	fmt.Printf("Opening device %s\n", *portPath)

	sensor, err := sds011.New(*portPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sensor.Close()

	fmt.Println("Checking if sensor is awake")
	err = sensor.Awake()
	if err != nil {
		log.Fatal(err)
	}
	did, err := sensor.DeviceID()
	fmt.Println(did)

	fmt.Printf("Setting %s to 1m readings\n", *portPath)
	err = sensor.SetCycle(1)
	if err != nil {
		log.Fatal(err)
	}

	mde, err := sensor.Firmware()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Firmware version: %s\n", mde)

	fmt.Printf("Activating %s\n", *portPath)
	err = sensor.MakeActive()
	if err != nil {
		log.Fatal(err)
	}

	pmcounters := expvar.NewMap("pmcounters")
	pm25 := new(expvar.Float)
	pm10 := new(expvar.Float)
	pmcounters.Set("pm2_5", pm25)
	pmcounters.Set("pm10", pm10)

	fmt.Printf("Exporting readings from %s on http://%s/debug/vars\n", *portPath, url)
	for {
		point, err := sensor.Get()
		if err != nil {
			log.Printf("ERROR: sensor.Get: %v", err)
			continue
		}
		pm25.Set(point.PM25)
		pm10.Set(point.PM10)
		if *debugFlag == true {
			fmt.Fprintf(os.Stdout, "%v, PM2.5: %3.2f,  PM10: %3.2f\n", point.Timestamp.Format(time.RFC3339), point.PM25, point.PM10)
		}
	}
}
