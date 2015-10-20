//Package symmsummary pulls out disk info from symmapi_db.bin.

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func check(function string, e error) {
	if e != nil {
		log.Fatal(function, e)
	}
}

func locateFile(exe string) string {
	progDirOld := `C:\Program Files (x86)\EMC\SYMCLI\bin\` + exe
	progDirNew := `C:\Program Files\EMC\SYMCLI\bin\` + exe
	fileLocation := ""

	if _, err := os.Stat(progDirNew); err == nil {
		fileLocation = progDirNew
	} else if _, err := os.Stat(progDirOld); err == nil {
		fileLocation = progDirOld
	} else {
		log.Fatal(exe + " is required, but is not found.\nLocations checked were:\n" + progDirNew + "\n" + progDirOld)
	}
	return fileLocation
}

func getVersion(exe string) string {
	cmd := exec.Command(exe, "-version")
	stdout, err := cmd.StdoutPipe()
	check("Version", err)

	versionLabel := regexp.MustCompile("Symmetrix CLI \\(SYMCLI\\) Version")
	version := "none"
	output := bufio.NewScanner(stdout)
	go func() {
		for output.Scan() {
			if versionLabel.MatchString(output.Text()) == true {
				version = strings.Split(strings.Split(output.Text(), ": ")[1], " (")[0]
				break
			}
		}
	}()
	err = cmd.Start()
	check("Version", err)
	err = cmd.Wait()
	check("Version", err)
	if version == "none" {
		log.Fatal("Unable to determine exe version\n")
	}

	return version
}

type Array struct {
	sid     string
	attach  string
	model   string
	mcode   string
	cache   int
	devs    int
	symdevs int
}

type Disk struct {
	Size  int
	Speed int
	Tech  string
}

type Worker struct {
	Cmd  string
	Args []string
}

func (cmd *Worker) Run() *bufio.Scanner {
	prep := exec.Command(cmd.Cmd, cmd.Args...)
	stdout, err := prep.StdoutPipe()
	check(cmd.Cmd, err)
	prep.Start()
	result := bufio.NewScanner(stdout)
	return (result)
}
func getMemory(exe string) <-chan string {
	outchan := make(chan string)
	go func() {
		command := &Worker{Cmd: exe, Args: []string{"list", "-memory"}}
		output := command.Run()
		for output.Scan() {
			outchan <- output.Text()
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
		command := &Worker{Cmd: exe, Args: []string{"list"}}
		output := command.Run()
		for output.Scan() {
			if label.MatchString(output.Text()) == true {
				arrayData := strings.Fields(output.Text())
				cacheInt, err := strconv.Atoi(arrayData[4])
				check("List", err)
				devsInt, err := strconv.Atoi(arrayData[5])
				check("List", err)
				symdevsInt, err := strconv.Atoi(arrayData[6])
				check("List", err)
				arrays[arrayData[0]] = Array{
					sid:     arrayData[0],
					attach:  arrayData[1],
					model:   arrayData[2],
					mcode:   arrayData[3],
					cache:   cacheInt,
					devs:    devsInt,
					symdevs: symdevsInt}
				break
			}
		}
		for _, a := range arrays {
			outchan <- fmt.Sprintf("Serial Number: %s	Model: %s	Microcode: %s	Cache: %d\n", a.sid, a.model, a.mcode, a.cache)
		}
		close(outchan)
	}()
	return outchan
}

func getCabinets(exe string) <-chan string {
	outchan := make(chan string)
	go func() {
		command := &Worker{Cmd: exe, Args: []string{"list", "-bay_info"}}
		output := command.Run()
		for output.Scan() {
			outchan <- output.Text()
		}
		close(outchan)
	}()
	return outchan
}

func getDisks(exe string) <-chan string {
	targetLabel := regexp.MustCompile("Target ID")
	techLabel := regexp.MustCompile("Technology")
	speedLabel := regexp.MustCompile("Speed")
	sizeLabel := regexp.MustCompile("Total Disk Capacity \\(MB\\)")
	outchan := make(chan string)
	disks := make(map[string]int)
	var size string
	var speed string
	var tech string
	go func() {
		command := &Worker{Cmd: exe, Args: []string{"list", "-v"}}
		output := command.Run()
		for output.Scan() {
			outputText := output.Text()
			if targetLabel.MatchString(outputText) {
				tech = "X"
				speed = "X"
				size = "X"
			} else if techLabel.MatchString(outputText) {
				tech = strings.TrimSpace(strings.Split(outputText, ":")[1])
			} else if speedLabel.MatchString(outputText) {
				speed = cleanSpeed(strings.TrimSpace(strings.Split(outputText, ":")[1]))
			} else if sizeLabel.MatchString(outputText) {
				size = strings.TrimSpace(strings.Split(outputText, ":")[1])
				if size == "0" {
					size = "X"
					speed = "X"
					tech = "X"
				} else {
					size=cleanSize(size)
				}
			}

			if tech != "X" && speed != "X" && size != "X" {
				disks[size+" "+speed+" "+tech]=disks[size+" "+speed+" "+tech] +  1
				size = "X"
				speed = "X"
				tech = "X"
			}
		}
		for d := range disks {
			outchan <- fmt.Sprintf("%s %d\n", d, disks[d])
		}
		
		close(outchan)
	}()
	return outchan
}

func cleanSize(size string) string {
	num,err:=strconv.Atoi(size)
	check("Clean size: ", err)
	var newsize int
	if num < 36384 {
		newsize=36
	} else if num > 36384 && num < 74752 {
		newsize=73
	} else if num > 74752 && num < 102400 {
		newsize=100
	} else if num > 102400 && num < 149504 {
		newsize=146
	} else if num > 149504 && num < 204800 {
		newsize=200
	} else if num > 204800 && num < 307200 {
		newsize=300
	} else if num > 307200 && num < 409600 {
		newsize=400
	} else if num > 409600 && num < 460800 {
		newsize=450
	} else if num > 460800 && num < 512000 {
		newsize=500
	} else if num > 512000 && num < 614400 {
		newsize=600
	} else if num > 611400 && num < 768000 {
		newsize=750
	} else if num > 768000 && num < 1024000 {
		newsize=1000
	} else if num > 1024000 && num < 2048000 {
		newsize=2000
	} else if num > 2048000 && num < 3072000 {
		newsize=3000
	} else {
		newsize=-1
	}
	return strconv.Itoa(newsize)
}

func cleanMemorySize(size string) string {
	var newsize string
	
	size16GB:=regexp.MustCompile("16384")
	size32GB:=regexp.MustCompile("(28672|32768)")
	size64GB:=regexp.MustCompile("65536")
	size128GB:=regexp.MustCompile("131072")
	if size16GB.MatchString(size) {
		newsize="16GB"
	} else if size32GB.MatchString(size) {
		newsize="32GB"
	} else if size64GB.MatchString(size) {
		newsize="64GB"
	} else if size128GB.MatchString(size) {
		newsize="128GB"
	} 
	
	return newsize
}

func cleanSpeed(speed string) string {
	var newspeed string
	speed15k:=regexp.MustCompile("15000")
	speed10k:=regexp.MustCompile("10000")
	speed7200:=regexp.MustCompile("7200")
	speedEFD:=regexp.MustCompile("^0$")
	
	if speed15k.MatchString(speed) {
		newspeed="15k"
	} else if speed10k.MatchString(speed) {
		newspeed="10k"
	} else if speed7200.MatchString(speed) {
		newspeed="7.2k"
	} else if speedEFD.MatchString(speed) {
		newspeed="EFD"
	} 
	return newspeed
}
func main() {
	fileName := "symapi_db.bin"
	os.Setenv("SYMCLI_OFFLINE", "1")
	os.Setenv("SYMCLI_DB_FILE", fileName)

	requiredVersion := "V7.4.0"
	diskexe := locateFile("symdisk.exe")
	cfgexe := locateFile("symcfg.exe")
	version := getVersion(cfgexe)
	if requiredVersion > version {
		log.Fatal(requiredVersion + " is required. Installed version is " + version)
	}

	symms := getSymmList(cfgexe)
	memory := getMemory(cfgexe)
	cabinets := getCabinets(cfgexe)
	disks := getDisks(diskexe)

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
}
