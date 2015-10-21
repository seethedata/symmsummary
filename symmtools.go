//Package symmsummary pulls out disk info from symmapi_db.bin.

package symmtools

import (
	"bufio"
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

func LocateFile(exe string) string {
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

func GetVersion(exe string) string {
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
