package common

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

// GetAppConfigPath returns the path of the executable with extension .config
func getAppConfigPath() string {
	nameOfExe := os.Args[0]
	ext := path.Ext(nameOfExe)
	configPath := nameOfExe[0:len(nameOfExe)-len(ext)] + ".config"
	return configPath
}

func GetConfiguration(config interface{}, path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {return err}

	err = json.Unmarshal(data, &config)
	return err
}

func SaveConfiguration(conf interface{}, path string) error {
	data, err := json.MarshalIndent(conf, "", "    ")
	if err == nil {
		err = ioutil.WriteFile(path, data, 0644)
	}
	return err
}

func GetAppConfiguration(config interface{}) error {
	return GetConfiguration(config, getAppConfigPath())
}

func SaveAppConfiguration(config interface{}) error {
	return SaveConfiguration(config, getAppConfigPath())
}

