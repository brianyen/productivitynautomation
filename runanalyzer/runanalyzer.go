package main

// A simple last 3 builds abort jobs analyzer
// jagadesh.munta@couchbase.com

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strings"
)

// N1QLQryResult type
type N1QLQryResult struct {
	Status  string
	Results []N1QLResult
}

// N1QLResult type
type N1QLResult struct {
	Aname    string
	JURL     string
	URLbuild int64
}

// TotalCycleTimeQryResult type
type TotalCycleTimeQryResult struct {
	Status  string
	Results []TotalCycleTime
}

// TotalCycleTime type
type TotalCycleTime struct {
	Totaltime int64
}

const url = "http://172.23.109.245:8093/query/service"

func main() {
	fmt.Println("*** Helper Tool ***")
	action := flag.String("action", "usage", usage())
	flag.Parse()

	if *action == "lastaborted" {
		lastabortedjobs()
	} else if *action == "savejoblogs" {
		savejoblogs()
	} else if *action == "totalduration" {
		fmt.Println("Total duration: ", gettotalbuildcycleduration(os.Args[3]))
	} else if *action == "runquery" {
		fmt.Println("Query Result: ", runquery(os.Args[3]))
	} else if *action == "usage" {
		fmt.Println(usage())
	}
}

func usage() string {
	fileName, _ := os.Executable()
	return "Usage: " + fileName + " -h | --help \nEnter action value. \n" +
		"-action lastaborted 6.5.0-4106 6.5.0-4059 6.5.0-4000  : to get the aborted jobs common across last 3 builds\n" +
		"-action savejoblogs 6.5.0-4106  : to get the jenkins logs for a given build\n" +
		"-action totalduration 6.5.0-4106  : to get the total time duration for a build cyle\n" +
		"-action runquery 'select * from server where lower(`os`)=\"centos\" and `build`=\"6.5.0-4106\"' : to run a given query statement"
}
func runquery(qry string) string {
	//url := "http://172.23.109.245:8093/query/service"
	fmt.Println("ACTION: runquery")
	fmt.Println("query=" + qry)
	localFileName := "qryresult.json"
	if err := executeN1QLStmt(localFileName, url, qry); err != nil {
		panic(err)
	}

	resultFile, err := os.Open(localFileName)
	if err != nil {
		fmt.Println(err)
	}
	defer resultFile.Close()

	byteValue, _ := ioutil.ReadAll(resultFile)
	return string(byteValue)
}

func gettotalbuildcycleduration(buildN string) string {
	fmt.Println("action: totalduration")

	//url := "http://172.23.109.245:8093/query/service"
	qry := "select sum(duration) as totaltime from server b where lower(b.os)=\"centos\" and b.`build`=\"" + buildN + "\""
	fmt.Println("query=" + qry)
	localFileName := "duration.json"
	if err := executeN1QLStmt(localFileName, url, qry); err != nil {
		panic(err)
	}

	resultFile, err := os.Open(localFileName)
	if err != nil {
		fmt.Println(err)
	}
	defer resultFile.Close()

	byteValue, _ := ioutil.ReadAll(resultFile)

	var result TotalCycleTimeQryResult

	err = json.Unmarshal(byteValue, &result)
	var ttime string
	if result.Status == "success" {
		fmt.Println("Total time in millis: ", result.Results[0].Totaltime)

		hours := math.Floor(float64(result.Results[0].Totaltime) / 1000 / 60 / 60)
		secs := result.Results[0].Totaltime % (1000 * 60 * 60)
		mins := math.Floor(float64(secs) / 60 / 1000)
		secs = result.Results[0].Totaltime * 1000 % 60
		fmt.Printf("%02d hrs : %02d mins :%02d secs", int64(hours), int64(mins), int64(secs))
		//ttime = string(hours) + ": " + string(mins) + ": " + string(secs)
	} else {
		fmt.Println("Status: Failed")
	}

	return ttime

}

func savejoblogs() {
	fmt.Println("action: savejoblogs")
	var build1 string
	if len(os.Args) < 2 {
		fmt.Println("Enter the build to save the jenkins job logs.")
		os.Exit(1)
	} else {
		build1 = os.Args[3]
	}

	//url := "http://172.23.109.245:8093/query/service"
	qry := "select b.name as aname,b.url as jurl,b.build_id urlbuild from server b where lower(b.os)=\"centos\" and b.`build`=\"" + build1 + "\""
	fmt.Println("query=" + qry)
	localFileName := "result.json"
	if err := executeN1QLStmt(localFileName, url, qry); err != nil {
		panic(err)
	}

	resultFile, err := os.Open(localFileName)
	if err != nil {
		fmt.Println(err)
	}
	defer resultFile.Close()

	byteValue, _ := ioutil.ReadAll(resultFile)

	var result N1QLQryResult

	err = json.Unmarshal(byteValue, &result)
	//fmt.Println("Status=" + result.Status)
	//fmt.Println(err)
	f, err := os.Create("all_jobs.csv")
	defer f.Close()

	w := bufio.NewWriter(f)

	if result.Status == "success" {
		fmt.Println("Count: ", len(result.Results))
		for i := 0; i < len(result.Results); i++ {
			//fmt.Println((i + 1), result.Results[i].Aname, result.Results[i].JURL, result.Results[i].URLbuild)
			fmt.Print(strings.TrimSpace(result.Results[i].Aname), ",", strings.TrimSpace(result.Results[i].JURL), ",",
				result.Results[i].URLbuild, "\n")
			_, err = fmt.Fprintf(w, "%s,%s,%d\n", strings.TrimSpace(result.Results[i].Aname), strings.TrimSpace(result.Results[i].JURL),
				result.Results[i].URLbuild)
		}
		w.Flush()

	} else {
		fmt.Println("Status: Failed")
	}
}

func lastabortedjobs() {
	fmt.Println("action: lastaborted")
	var build1 string
	var build2 string
	var build3 string
	if len(os.Args) < 4 {
		fmt.Println("Enter the last 3 builds and first being the latest.")
		os.Exit(1)
	} else {
		build1 = os.Args[3]
		build2 = os.Args[4]
		build3 = os.Args[5]
	}

	//url := "http://172.23.109.245:8093/query/service"
	qry := "select b.name as aname,b.url as jurl,b.build_id urlbuild from server b where lower(b.os)=\"centos\" and b.result=\"ABORTED\" and b.`build`=\"" + build1 + "\" and b.name in (select raw a.name from server a where lower(a.os)=\"centos\" and a.result=\"ABORTED\" and a.`build`=\"" + build2 + "\" intersect select raw name from server where lower(os)=\"centos\" and result=\"ABORTED\" and `build`=\"" + build3 + "\" intersect select raw name from server where lower(os)=\"centos\" and result=\"ABORTED\" and `build`=\"" + build1 + "\")"
	fmt.Println("query=" + qry)
	localFileName := "result.json"
	if err := executeN1QLStmt(localFileName, url, qry); err != nil {
		panic(err)
	}

	resultFile, err := os.Open(localFileName)
	if err != nil {
		fmt.Println(err)
	}
	defer resultFile.Close()

	byteValue, _ := ioutil.ReadAll(resultFile)

	var result N1QLQryResult

	err = json.Unmarshal(byteValue, &result)
	//fmt.Println("Status=" + result.Status)
	//fmt.Println(err)
	f, err := os.Create("aborted_jobs.csv")
	defer f.Close()

	w := bufio.NewWriter(f)
	if result.Status == "success" {
		fmt.Println("Count: ", len(result.Results))
		for i := 0; i < len(result.Results); i++ {
			//fmt.Println((i + 1), result.Results[i].Aname, result.Results[i].JURL, result.Results[i].URLbuild)
			fmt.Print(strings.TrimSpace(result.Results[i].Aname), "\t", strings.TrimSpace(result.Results[i].JURL), "\t",
				result.Results[i].URLbuild)
			_, err = fmt.Fprintf(w, "%s,%s,%d\n", strings.TrimSpace(result.Results[i].Aname), strings.TrimSpace(result.Results[i].JURL),
				result.Results[i].URLbuild)
		}
		w.Flush()

	} else {
		fmt.Println("Status: Failed")
	}
}

// DownloadFile2 will download the given url to the given file.
func executeN1QLStmt(localFilePath string, remoteURL string, statement string) error {

	req, err := http.NewRequest("GET", remoteURL, nil)
	if err != nil {
		return err
	}
	urlq := req.URL.Query()
	urlq.Add("statement", statement)
	req.URL.RawQuery = urlq.Encode()
	u := req.URL.String()
	//fmt.Println(req.URL.String())
	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(localFilePath)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, resp.Body)
	return err
}

// DownloadFile will download the given url to the given file.
func DownloadFile(localFilePath string, remoteURL string) error {
	resp, err := http.Get(remoteURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(localFilePath)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, resp.Body)
	return err
}
