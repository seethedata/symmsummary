//Package symmsummary pulls out disk info from symmapi_db.bin.

package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"github.com/seethedata/symmtools"
)

func check(function string, e error) {
	if e != nil {
		log.Fatal(function, e)
	}
}

type Array struct {
	sid     string
	attach  string
	model   string
	mcode   string
	cache   string
	devs    int
	symdevs int
}

type Disk struct {
	Size  int
	Speed int
	Tech  string
}


func getMemory(exe string) <-chan string {
	memPattern := regexp.MustCompile("[0-9]{5,6}$")
	outchan := make(chan string)
	go func() {
		command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-memory"}}
		output := command.Run()
		for output.Scan() {
			outputText := output.Text()
			if memPattern.MatchString(outputText) {
				outputText = cleanMemorySize(outputText)
			}
			outchan <- outputText
		}
		close(outchan)
	}()
	return outchan
}

func getSymmList(exe string) <-chan string {
	label := regexp.MustCompile("(DMX|VMAX)")
	outchan := make(chan string)
	arrays := make(map[string]Array)
	go func() {
		command := &symmtools.Worker{Cmd: exe, Args: []string{"list"}}
		output := command.Run()
		for output.Scan() {
			if label.MatchString(output.Text()) == true {
				arrayData := strings.Fields(output.Text())
				cache:=arrayData[4]
				devsInt, err := strconv.Atoi(arrayData[5])
				check("List", err)
				symdevsInt, err := strconv.Atoi(arrayData[6])
				check("List", err)
				arrays[arrayData[0]] = Array{
					sid:     arrayData[0],
					attach:  arrayData[1],
					model:   arrayData[2],
					mcode:   arrayData[3],
					cache:   cache,
					devs:    devsInt,
					symdevs: symdevsInt}
				break
			}
		}
		for _, a := range arrays {
			outchan <- fmt.Sprintf("Serial Number: %s	Model: %s	Microcode: %s	Cache: %s\n", a.sid, a.model, a.mcode, cleanMemorySize(a.cache))
		}
		close(outchan)
	}()
	return outchan
}

func getCabinets(exe string) <-chan string {
	cabinetLabel:=regexp.MustCompile("Bay Location")
	outchan := make(chan string)
	go func() {
		command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-bay_info"}}
		output := command.Run()
		for output.Scan() {
			outputText := output.Text()
			if cabinetLabel.MatchString(outputText) {
				outchan <- outputText
			}
		}
		close(outchan)
	}()
	return outchan
}

func getPools(exe string) <-chan string {
	skipLabel:=regexp.MustCompile("Symmetrix ID")
	outchan := make(chan string)
	go func() {
		command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-thin", "-pool", "-gb"}}
		output := command.Run()
		for output.Scan() {
			outputText := output.Text()
			if ! skipLabel.MatchString(outputText) {
				outchan <- outputText
			}
		}
		close(outchan)
	}()
	return outchan
}

func getSoftware(exe string) <-chan string {
	featureLabel:=regexp.MustCompile("FeatureName")
	capTypeLabel:=regexp.MustCompile("FeatureCapacityType")
	featureCapLabel:=regexp.MustCompile("FeatureCapacity")
	whiteSpace := regexp.MustCompile("\\s")
	outchan := make(chan string)
	featureName:="X"
	featureType:="X"
	featureCap:="X"
	go func() {
		command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-features", "-enabled"}}
		output := command.Run()
		for output.Scan() {
			outputText := whiteSpace.ReplaceAllString(output.Text(), "")
			if featureLabel.MatchString(outputText) {
				featureName= strings.Split(outputText, ":")[1]
			} else if capTypeLabel.MatchString(outputText) {
				featureType	= strings.Split(outputText, ":")[1]
			} else if featureCapLabel.MatchString(outputText) {
				if featureType == "TBofTotalCapacity" {
					featureCap= ": " + strings.Split(outputText, ":")[1]
				} else {
					featureCap=""
				}
			}
		if featureName != "X" && featureType != "X" && featureCap !="X"{
			outchan <- fmt.Sprintf("%s %s%s\n", featureName, featureType, featureCap)
			featureName, featureType,featureCap="X","X","X"
		}
		}
		
		close(outchan)
	}()
	return outchan
}
func getDisks(exe string) <-chan string {
	targetLabel := regexp.MustCompile("(TargetID|Director)")
	techLabel := regexp.MustCompile("Technology")
	speedLabel := regexp.MustCompile("Speed")
	sizeLabel := regexp.MustCompile("(TotalDiskCapacity\\(MB\\)|RatedDiskCapacity\\(GB\\)|ActualDiskCapacity\\(MB\\))")
	hotspareSizeLabel := regexp.MustCompile("(RatedDiskCapacity\\(GB\\)|ActualDiskCapacity\\(MB\\))")
	whiteSpace := regexp.MustCompile("\\s")
	outchan := make(chan string)
	disks := make(map[string]int)
	hsdisks := make(map[string]int)
	var size string
	var speed string
	var tech string
	go func() {
		command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-v"}}
		output := command.Run()
		tech = "X"
		speed = "X"
		size = "X"
		for output.Scan() {
			outputText := whiteSpace.ReplaceAllString(output.Text(), "")
			if targetLabel.MatchString(outputText) {
				tech = "X"
				speed = "X"
				size = "X"
			} else if techLabel.MatchString(outputText) {
				tech = strings.Split(outputText, ":")[1]
			} else if speedLabel.MatchString(outputText) {
				speed = cleanSpeed(strings.Split(outputText, ":")[1])
			} else if sizeLabel.MatchString(outputText) {
				size = strings.Split(outputText, ":")[1]
				if size == "0" {
					size = "X"
					speed = "X"
					tech = "X"
				} else {
					size = cleanSize(size)
				}
			}

			if tech != "X" && speed != "X" && size != "X" {
				diskType := size + " " + speed + " " + tech + " "
				disks[diskType] += 1
				size = "X"
				speed = "X"
				tech = "X"
			}
		}
		for d := range disks {
			outchan <- fmt.Sprintf("(%d) %s\n", disks[d], d)
		}
		
		outchan<-"-----------HotSpares-----------"
		command= &symmtools.Worker{Cmd: exe, Args: []string{"list", "-hotspares", "-v"}}
		output = command.Run()
		tech = "X"
		speed = "X"
		size = "X"
		for output.Scan() {
			outputText := whiteSpace.ReplaceAllString(output.Text(), "")
			if targetLabel.MatchString(outputText) {
				tech = "X"
				speed = "X"
				size = "X"
			} else if techLabel.MatchString(outputText) {
				tech = strings.Split(outputText, ":")[1]
			} else if speedLabel.MatchString(outputText) {
				speed = cleanSpeed(strings.Split(outputText, ":")[1])
			} else if hotspareSizeLabel.MatchString(outputText) {
				size = strings.Split(outputText, ":")[1]
				if size == "0" {
					size = "X"
					speed = "X"
					tech = "X"
				} else {
					size = size
				}
			}
			
			if tech != "X" && speed != "X" && size != "X" {
				diskType := size + " " + speed + " " + tech + " "
				hsdisks[diskType] += 1
				size = "X"
				speed = "X"
				tech = "X"
			}
		}

		for d := range hsdisks {
			outchan <- fmt.Sprintf("(%d) %s\n", hsdisks[d], d)
		}
		close(outchan)
	}()
	return outchan
}

func cleanSize(size string) string {
	num, err := strconv.Atoi(size)
	check("Clean size: ", err)
	var newsize int
	if num < 36384 {
		newsize = 36
	} else if num > 36384 && num < 74752 {
		newsize = 73
	} else if num > 74752 && num < 102400 {
		newsize = 100
	} else if num > 102400 && num < 149504 {
		newsize = 146
	} else if num > 149504 && num < 204800 {
		newsize = 200
	} else if num > 204800 && num < 307200 {
		newsize = 300
	} else if num > 307200 && num < 409600 {
		newsize = 400
	} else if num > 409600 && num < 460800 {
		newsize = 450
	} else if num > 460800 && num < 512000 {
		newsize = 500
	} else if num > 512000 && num < 614400 {
		newsize = 600
	} else if num > 611400 && num < 768000 {
		newsize = 750
	} else if num > 768000 && num < 1024000 {
		newsize = 1000
	} else if num > 1024000 && num < 2048000 {
		newsize = 2000
	} else if num > 2048000 && num < 3072000 {
		newsize = 3000
	} else {
		newsize = num
	}
	return strconv.Itoa(newsize)
}

func cleanMemorySize(size string) string {
	var newsize string

	size16GB := regexp.MustCompile("16384")
	size32GB := regexp.MustCompile("(28672|32768)")
	size64GB := regexp.MustCompile("(60160|65536)")
	size128GB := regexp.MustCompile("131072")
	size256GB := regexp.MustCompile("240640")
	if size16GB.MatchString(size) {
		newsize = size16GB.ReplaceAllString(size, "16GB")
	} else if size32GB.MatchString(size) {
		newsize = size32GB.ReplaceAllString(size, "32GB")
	} else if size64GB.MatchString(size) {
		newsize = size64GB.ReplaceAllString(size, "64GB")
	} else if size128GB.MatchString(size) {
		newsize = size128GB.ReplaceAllString(size, "128GB")
	} else if size256GB.MatchString(size) {
		newsize = size256GB.ReplaceAllString(size, "256GB")
	} else {
		newsize=size
	}
	return newsize
}

func cleanSpeed(speed string) string {
	var newspeed string
	speed15k := regexp.MustCompile("15000")
	speed10k := regexp.MustCompile("10000")
	speed7200 := regexp.MustCompile("7200")
	speedEFD := regexp.MustCompile("^0$")

	if speed15k.MatchString(speed) {
		newspeed = "15k"
	} else if speed10k.MatchString(speed) {
		newspeed = "10k"
	} else if speed7200.MatchString(speed) {
		newspeed = "7.2k"
	} else if speedEFD.MatchString(speed) {
		newspeed = "EFD"
	}
	return newspeed
}
func main() {
	fileName := "symapi_db.bin"
	os.Setenv("SYMCLI_OFFLINE", "1")
	os.Setenv("SYMCLI_DB_FILE", fileName)

	requiredVersion := "V7.4.0"
	diskexe := symmtools.LocateFile("symdisk.exe")
	cfgexe := symmtools.LocateFile("symcfg.exe")
	version := symmtools.GetVersion(cfgexe)
	if requiredVersion > version {
		log.Fatal(requiredVersion + " is required. Installed version is " + version)
	}

	symms := getSymmList(cfgexe)
	memory := getMemory(cfgexe)
	cabinets := getCabinets(cfgexe)
	disks := getDisks(diskexe)
	pools := getPools(cfgexe)
	software :=getSoftware(cfgexe)
	fmt.Println("-------------Symm--------------")
	for output := range symms {
		fmt.Printf("%s\n", output)
	}
	fmt.Println("-------------Memory--------------")
	for output := range memory {
		fmt.Printf("%s\n", output)
	}

	fmt.Println("-------------Cabinets--------------")
	for output := range cabinets {
		fmt.Printf("%s\n", output)
	}

	fmt.Println("-------------Disks--------------")
	for output := range disks {
		fmt.Printf("%s\n", output)
	}
	
	fmt.Println("-------------Pools--------------")
	for output := range pools {
		fmt.Printf("%s\n", output)
	}
	
	fmt.Println("-------------Software (Experimental - May not be accurate)--------------")
	for output := range software {
		fmt.Printf("%s\n", output)
	}
}
