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

type Worker struct {
	Command string
	Args    string
	Output  chan string
}

func (cmd *Worker) Run() {
	prep := exec.Command(cmd.Command, cmd.Args)
	stdout, err := prep.StdoutPipe()
	check("Command: \""+cmd.Command+" "+cmd.Args+"\" ", err)
	prep.Start()
	out := bufio.NewScanner(stdout)
	for out.Scan() {
		cmd.Output <- string(out.Text())
	}
	err = prep.Wait()
	check("Command: \""+cmd.Command+" "+cmd.Args+"\" ", err)
}

func main() {
	fileName := "symapi_db.bin"
	os.Setenv("SYMCLI_OFFLINE", "1")
	os.Setenv("SYMCLI_DB_FILE", fileName)

	requiredVersion := "V7.4.0"
	//diskexe := locateFile("symdisk.exe")
	cfgexe := locateFile("symcfg.exe")
	version := getVersion(cfgexe)
	if requiredVersion > version {
		log.Fatal(requiredVersion + " is required. Installed version is " + version)
	}

	listch := make(chan string)

	fmt.Println("-------------Symm--------------")

	label := regexp.MustCompile("(DMX|VMAX)")

	arrays := make(map[string]Array)
	getSymList := &Worker{Command: cfgexe, Args: "list", Output: listch}
	go getSymList.Run()
	for output := range listch {
		if label.MatchString(output) == true {
			arrayData := strings.Fields(output)
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
		fmt.Printf("Serial Number: %s	Model: %s	Microcode: %s	Cache: %d\n", a.sid, a.model, a.mcode, a.cache)
	}
	memch := make(chan string)

	fmt.Println("-------------Memory--------------")
	getMemory := &Worker{Command: cfgexe, Args: "list -memory", Output: memch}
	go getMemory.Run()
	for output := range memch {
		fmt.Println(output)
	}
}
