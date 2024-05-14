package apps

import (
	"os"

	"gopkg.in/yaml.v3"
)

type config struct {
	SdkPath      string `yaml:"sdk_path"`
	Keystore     string `yaml:"keystore"`
	Keystorepass string `yaml:"keystorepass"`
	Keypass      string `yaml:"keypass"`
	Alias        string `yaml:"repo_keyalias"`
}

func ParseFdroidConfig(filepath string) (c *config, err error) {
	f, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&c)
	return
}
