package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"
)

type flags struct {
	url url.URL
}

type ipc struct {
	Count     int64
	Time      time.Time
	TimeCount int64
}

type ips map[string]ipc

type counter struct {
	sync.Mutex
	IpCount ips
}

var Dat *counter

func init() {
	Dat = new(counter)
	Dat.IpCount = make(map[string]ipc)
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU()*2 + 1

	f, err := getFlags()
	if err != nil {
		log.Fatalf("flags parsing fail: %v", err)
	}

	http.HandleFunc("/gazer/logip", logIpHandler)

	err = http.ListenAndServe(getPort(f.url), nil)
	if err != nil {
		log.Fatalf("ListenAndServe: ", err)
	}
}

func getFlags() (flags, error) {

	u := flag.String("url", "http://localhost:8080", "catcher url")

	flag.Parse()

	ur, err := url.Parse(*u)
	if err != nil {
		log.Printf("url parse err: %v", err)
		return flags{}, err
	}

	return flags{*ur}, nil
}

func logIpHandler(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		logf("err: r.ParseForm: %v", err)
		return
	}

	fmt.Printf("%v ms, %v -> %v, Ips %v\n",
		time.Now().UnixNano()/1000000,
		countIP(r.FormValue("src")),
		countIP(r.FormValue("dst")),
		len(Dat.IpCount))
}

func countIP(ipp string) string {

	if n := strings.Index(ipp, ":"); n != -1 {
		ipp = ipp[:n]
	}

	decideInformer(ipp)

	incIpCount(ipp)

	s := fmt.Sprintf("%v(%v)", ipp, getIpCount(ipp))

	return s
}

func getIpCount(ip string) int64 {

	Dat.Lock()
	defer Dat.Unlock()

	if _, ok := Dat.IpCount[ip]; ok == false {
		Dat.IpCount[ip] = ipc{0, time.Now(), 0}
	}

	return Dat.IpCount[ip].Count
}

func incIpCount(ip string) {

	Dat.Lock()
	defer Dat.Unlock()

	if _, ok := Dat.IpCount[ip]; ok == false {
		Dat.IpCount[ip] = ipc{0, time.Now(), 0}
	}

	if time.Since(Dat.IpCount[ip].Time).Seconds() > 60 {

		Dat.IpCount[ip] = ipc{
			Dat.IpCount[ip].Count + 1,
			time.Now(),
			Dat.IpCount[ip].Count + 1}
	} else {

		Dat.IpCount[ip] = ipc{
			Dat.IpCount[ip].Count + 1,
			Dat.IpCount[ip].Time,
			Dat.IpCount[ip].TimeCount}
	}
}

func decideInformer(ip string) {

	Dat.Lock()
	defer Dat.Unlock()

	if _, ok := Dat.IpCount[ip]; ok == false {
		go callInformer(ip)
	} else {

		if strings.Contains(ip, "192.168.1.22") == false {

			fmt.Printf("--------- ip: %v, sec: %v, delta: %v\n",
				ip,
				int(time.Since(Dat.IpCount[ip].Time).Seconds()),
				Dat.IpCount[ip].Count-Dat.IpCount[ip].TimeCount)

			if Dat.IpCount[ip].Count-Dat.IpCount[ip].TimeCount > 40 &&
				time.Since(Dat.IpCount[ip].Time).Seconds() < 60 {

				go callInformer(ip)

				Dat.IpCount[ip] = ipc{
					Dat.IpCount[ip].Count,
					time.Now(),
					Dat.IpCount[ip].Count}
			}
		}
	}
}

func callInformer(ip string) {

	_, err := http.PostForm("http://192.168.1.32:8082/message",
		url.Values{"message": {ip}})

	if err != nil {
		log.Printf("err: http.PostForm: %v", err)
	}
}

func getPort(u url.URL) string {

	r := u.Host

	if n := strings.Index(r, ":"); n != -1 {
		r = r[n:]
	} else {
		r = ":8080"
	}

	return r
}

func logf(f string, v ...interface{}) {
	s := fmt.Sprintf(f, v...)
	log.Printf(s)
}
