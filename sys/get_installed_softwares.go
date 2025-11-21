package main

import (
	"fmt"
	"golang.org/x/sys/windows/registry"
	"strings"
)

type Program struct {
	Name        string
	Version     string
	Publisher   string
	InstallLocation string
	InstallDate string
}

var (
	Microsoft = "Microsoft"
)

func getInstalledPrograms() ([]Program, error) {
	var programs []Program

	// Registry paths to check
	paths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	for _, path := range paths {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			return nil, fmt.Errorf("error opening registry key: %v", err)
		}
		defer k.Close()

		subkeys, err := k.ReadSubKeyNames(-1)
		if err != nil {
			return nil, fmt.Errorf("error reading subkeys: %v", err)
		}

		for _, subkey := range subkeys {
			program, err := readProgramInfo(path, subkey)
			if err == nil && program.Name != "" {
				if strings.HasPrefix(program.Publisher, Microsoft) {
					continue
				}
				programs = append(programs, program)
			}
		}
	}

	return programs, nil
}

func readProgramInfo(path, subkey string) (Program, error) {
	var program Program

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path+`\`+subkey, registry.QUERY_VALUE)
	if err != nil {
		return program, err
	}
	defer k.Close()

	name, _, err := k.GetStringValue("DisplayName")
	if err == nil {
		program.Name = name
	}

	version, _, err := k.GetStringValue("DisplayVersion")
	if err == nil {
		program.Version = version
	}

	publisher, _, err := k.GetStringValue("Publisher")
	if err == nil {
		program.Publisher = publisher
	}

	installDate, _, err := k.GetStringValue("InstallDate")
	if err == nil {
		program.InstallDate = installDate
	}

	installLocation, _, err:= k.GetStringValue("InstallLocation")
	if err != nil {
		installLocation, _, err = k.GetStringValue("UninstallString")
	}
	if installLocation != "" {
		program.InstallLocation = installLocation
	} 

	return program, nil
}

func main() {
	programs, err := getInstalledPrograms()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	for _, program := range programs {
		fmt.Printf("Name: %s\n", program.Name)
		if program.Version != "" {
			fmt.Printf("Version: %s\n", program.Version)
		}
		if program.Publisher != "" {
			fmt.Printf("Publisher: %s\n", program.Publisher)
		}
		if program.InstallDate != "" {
			fmt.Printf("Install Date: %s\n", program.InstallDate)
		}
		if program.InstallLocation!= "" {
			fmt.Printf("Install Location: %s\n", program.InstallLocation)
		}
		fmt.Println(strings.Repeat("-", 50))
	}
}
