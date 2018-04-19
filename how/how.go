package how

import (
	"bufio"
	"errors"
	"flag"
	"os"
	"path"
	"reflect"
	"strings"

	"webServer/logme"
)

// small package for a simple 'key = value' config file, with optional overriding via command line flags

var (
	ShowingHelp = errors.New("showing help")
	showHelp    = false
)

type Config struct {
	flags *flag.FlagSet
}

func (cfg *Config) Load(name string, config interface{}) error {
	configT := reflect.TypeOf(config)
	// dereference until we get the concrete type
	for configT.Kind() == reflect.Ptr {
		configT = configT.Elem()
	}

	configI := reflect.ValueOf(config)
	configV := configI.Elem()

	if configV.Kind() != reflect.Struct {
		return errors.New("config is not a struct type")
	}

	cfg.flags = flag.NewFlagSet(name, flag.ContinueOnError)

	cfg.createFlags(configV, configT)

	err := cfg.loadConfigFile()
	if err != nil {
		return err
	}

	err = cfg.flags.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	if showHelp {
		cfg.flags.PrintDefaults()
		return ShowingHelp
	}

	return nil
}

func (cfg *Config) createFlags(configV reflect.Value, configT reflect.Type) {
	for i := 0; i < configV.NumField(); i++ {
		field := configV.Field(i)
		fieldT := configT.Field(i)
		tagsStr, ok := fieldT.Tag.Lookup("how")
		if !ok {
			continue
		}

		tags := strings.Split(tagsStr, ",")

		var (
			name      = "-" + tags[0]
			shortName = ""
			usage     = ""
		)

		if name == "-" {
			name += fieldT.Name
		}

		if len(tags) > 1 {
			shortName = tags[1]
		}

		if len(tags) > 2 {
			usage = tags[2]
		}

		switch field.Kind() {
		case reflect.Bool:
			cfg.flags.BoolVar(field.Addr().Interface().(*bool), name, field.Bool(), usage)

			if shortName != "" {
				cfg.flags.BoolVar(field.Addr().Interface().(*bool), shortName, field.Bool(), "alias for --"+name)
			}

		case reflect.Float64:
			cfg.flags.Float64Var(field.Addr().Interface().(*float64), name, field.Float(), usage)

			if shortName != "" {
				cfg.flags.Float64Var(field.Addr().Interface().(*float64), shortName, field.Float(), "alias for --"+name)
			}

		case reflect.Int:
			cfg.flags.IntVar(field.Addr().Interface().(*int), name, int(field.Int()), usage)

			if shortName != "" {
				cfg.flags.IntVar(field.Addr().Interface().(*int), shortName, int(field.Int()), "alias for --"+name)
			}

		case reflect.String:
			cfg.flags.StringVar(field.Addr().Interface().(*string), name, field.String(), usage)

			if shortName != "" {
				cfg.flags.StringVar(field.Addr().Interface().(*string), shortName, field.String(), "alias for --"+name)
			}

		case reflect.Uint:
			cfg.flags.UintVar(field.Addr().Interface().(*uint), name, uint(field.Uint()), usage)

			if shortName != "" {
				cfg.flags.UintVar(field.Addr().Interface().(*uint), shortName, uint(field.Uint()), "alias for --"+name)
			}

		default:
			logme.Warn().Printf("Unsupported type for config field '%s'\n", fieldT.Name)
		}
	}

	cfg.flags.BoolVar(&showHelp, "-help", false, "show this help message")
	cfg.flags.BoolVar(&showHelp, "h", false, "alias for --help")
}

func (cfg *Config) loadConfigFile() error {
	// find our config file
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		cfgDir = path.Join(os.Getenv("HOME"), ".config")
	}
	filepath := path.Join(cfgDir, "web", "web.conf")

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		// fallback to working directory
		filepath = "./web.conf"
	}

	fd, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}

	scan := bufio.NewScanner(fd)
	lineNum := 0

	for scan.Scan() {
		lineNum++
		line := strings.TrimSpace(scan.Text())

		// skip empty lines and comments
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		comps := strings.Split(line, "=")
		if len(comps) == 1 {
			logme.Warn().Printf("No setting done in %s:%d: '%s'\n", filepath, lineNum, line)
			continue
		}

		key, value := strings.TrimSpace(comps[0]), strings.TrimSpace(comps[1])

		cfgFlag := cfg.flags.Lookup(key)
		if cfgFlag == nil {
			logme.Warn().Printf("No such setting '%s' in %s:%d\n", key, filepath, lineNum)
			continue
		}

		err = cfgFlag.Value.Set(value)
		if err != nil {
			logme.Warn().Printf("Could not set '%s' to '%s' in %s:%d\n", key, value, filepath, lineNum)
		}
	}

	if scan.Err() != nil {
		return scan.Err()
	}

	return nil
}
