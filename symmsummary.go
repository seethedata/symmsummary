//Package symmsummary pulls out disk info from symmapi_db.bin.

package main

import (
	"fmt"
	"github.com/seethedata/symmtools"
	"log"
	"os"
	"regexp"
	"strings"
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

func getSymmList(exe string) <-chan string {
	label := regexp.MustCompile("(DMX|VMAX)")
	outchan := make(chan string)
	go func() {
		command := &symmtools.Worker{Cmd: exe, Args: []string{"list"}}
		output := command.Run()
		for output.Scan() {
			if label.MatchString(output.Text()) == true {
				arrayData := strings.Fields(output.Text())
				cache := arrayData[4]
				//devsInt, err := strconv.Atoi(arrayData[5])
				//check("List", err)
				//symdevsInt, err := strconv.Atoi(arrayData[6])
				//check("List", err)
				sid := arrayData[0]
				//attach:=  arrayData[1]
				model := arrayData[2]
				mcode := arrayData[3]
				//devs:=    devsInt
				//symdevs:= symdevsInt
				outchan <- fmt.Sprintf("%s	Model: %s	Microcode: %s	Cache: %s\n", sid, model, mcode, symmtools.CleanMemorySize(cache))
			}
		}
		close(outchan)
	}()
	return outchan
}

func getMemory(exe string, sids map[string]bool) <-chan string {
	memPattern := regexp.MustCompile("[0-9]{5,6}$")
	outchan := make(chan string)
	go func() {
		for sid := range sids {
			command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-memory", "-sid", sid}}
			output := command.Run()
			for output.Scan() {
				outputText := output.Text()
				if memPattern.MatchString(outputText) {
					outputText = symmtools.CleanMemorySize(outputText)
				}
				outchan <- outputText
			}
		}
		close(outchan)
	}()
	return outchan
}

func getCabinets(exe string, sids map[string]bool) <-chan string {
	cabinetLabel := regexp.MustCompile("Bay Location")
	outchan := make(chan string)
	go func() {
		for sid := range sids {
		outchan<-"--------------------" + sid + "--------------------"
			command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-bay_info", "-sid", sid}}
			output := command.Run()
			for output.Scan() {
				outputText := output.Text()
				if cabinetLabel.MatchString(outputText) {
					outchan <- outputText
				}
			}
		}
		close(outchan)
	}()
	return outchan
}

func getPools(exe string, sids map[string]bool) <-chan string {
	skipLabel := regexp.MustCompile("Symmetrix ID")
	outchan := make(chan string)
	go func() {
		for sid := range sids {
			outchan<-"--------------------" + sid + "--------------------"
			command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-thin", "-pool", "-gb", "-sid", sid}}
			output := command.Run()
			for output.Scan() {
				outputText := output.Text()
				if !skipLabel.MatchString(outputText) {
					outchan <- outputText
				}
			}
		}
		close(outchan)
	}()
	return outchan
}

func getSoftware(exe string, sids map[string]bool) <-chan string {
	featureLabel := regexp.MustCompile("FeatureName")
	capTypeLabel := regexp.MustCompile("FeatureCapacityType")
	featureCapLabel := regexp.MustCompile("FeatureCapacity")
	whiteSpace := regexp.MustCompile("\\s")
	outchan := make(chan string)
	featureName := "X"
	featureType := "X"
	featureCap := "X"
	go func() {
		for sid := range sids {
			command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-features", "-enabled", "-sid", sid}}
			output := command.Run()
			for output.Scan() {
				outputText := whiteSpace.ReplaceAllString(output.Text(), "")
				if featureLabel.MatchString(outputText) {
					featureName = strings.Split(outputText, ":")[1]
				} else if capTypeLabel.MatchString(outputText) {
					featureType = strings.Split(outputText, ":")[1]
				} else if featureCapLabel.MatchString(outputText) {
					if featureType == "TBofTotalCapacity" {
						featureCap = ": " + strings.Split(outputText, ":")[1]
					} else {
						featureCap = ""
					}
				}
				if featureName != "X" && featureType != "X" && featureCap != "X" {
					outchan <- fmt.Sprintf("%s %s%s\n", featureName, featureType, featureCap)
					featureName, featureType, featureCap = "X", "X", "X"
				}
			}
		}
		close(outchan)
	}()
	return outchan
}
func getDisks(exe string, sids map[string]bool) <-chan string {
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
		for sid := range sids {
			outchan<-"--------------------" + sid + "--------------------"
			command := &symmtools.Worker{Cmd: exe, Args: []string{"list", "-v","-sid",sid}}
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
					speed = symmtools.CleanSpeed(strings.Split(outputText, ":")[1])
				} else if sizeLabel.MatchString(outputText) {
					size = strings.Split(outputText, ":")[1]
					if size == "0" {
						size = "X"
						speed = "X"
						tech = "X"
					} else {
						size = symmtools.CleanSize(size)
					}
				}

				if tech != "X" && speed != "X" && size != "X" {
					diskType := size + " " + speed + " " + tech + " "
					disks[diskType]++
					size = "X"
					speed = "X"
					tech = "X"
				}
			}
			for d := range disks {
				outchan <- fmt.Sprintf("(%d) %s\n", disks[d], d)
			}

			outchan <- "-----------HotSpares-----------"
			command = &symmtools.Worker{Cmd: exe, Args: []string{"list", "-hotspares", "-v","-sid",sid}}
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
					speed = symmtools.CleanSpeed(strings.Split(outputText, ":")[1])
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
		}
		close(outchan)
	}()
	return outchan
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

	//software := getSoftware(cfgexe)
	sids := make(map[string]bool)

	fmt.Println("-------------Symm--------------")
	for symms := range getSymmList(cfgexe) {
		sid := strings.Fields(symms)[0]
		sids[sid] = true
		fmt.Printf("%s\n", symms)
	}

	fmt.Println("-------------Memory--------------")
	for memory := range getMemory(cfgexe, sids) {
		fmt.Printf("%s\n", memory)
	}

	fmt.Println("-------------Cabinets--------------")
	for cabinets := range getCabinets(cfgexe, sids) {
		fmt.Printf("%s\n", cabinets)
	}

	fmt.Println("-------------Disks--------------")
	for disks := range getDisks(diskexe, sids) {
		fmt.Printf("%s\n", disks)
	}

	fmt.Println("-------------Pools--------------")
	for pools := range getPools(cfgexe, sids) {
		fmt.Printf("%s\n", pools)
	}
	/*fmt.Println("-------------Software (Experimental - May not be accurate)--------------")
	for output := range software {
		fmt.Printf("%s\n", output)
	}*/
}
