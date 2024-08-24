package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/miekg/dns"
)

var config *Config
var configPath string
var defaultConfigPath = "~/.easydns/config.json"

type Record struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	Priority int    `json:"priority,omitempty"` // For MX and SRV records
	TTL      uint32 `json:"ttl,omitempty"`      // TTL for the record
}

type Records map[string]Record

// Config holds the DNS server configuration
type Config struct {
	Port    string  `json:"port"`
	Records Records `json:"records"`
}

var DefaultConfig = Config{
	Port: "53",
	Records: Records{
		"test.com": {
			Type:  "A",
			Value: "127.0.0.1",
			TTL:   600,
		},
		"www.test.com": {
			Type:  "CNAME",
			Value: "test.com",
			TTL:   600,
		},
		"mail.test.com": {
			Type:     "MX",
			Value:    "mail.somehost.com",
			Priority: 10,
			TTL:      60,
		},
	},
}

type ConfigNotFoundError struct {
	originalError error
}
type ConfigMalformedError struct {
	originalError error
}

func (e ConfigNotFoundError) Error() string {
	return fmt.Sprintf("config file not found: %v", e.originalError)
}

func (e ConfigMalformedError) Error() string {
	return fmt.Sprintf("config file is malformed: %v", e.originalError)
}

// LoadConfig reads and parses the JSON configuration file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, ConfigNotFoundError{originalError: err}
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, ConfigMalformedError{originalError: err}
	}
	return &config, nil
}

// handleDNSRequest handles incoming DNS queries
func handleDNSRequest(records Records) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		log.Printf("query: %s, from: %s", r.Question[0].Name, w.RemoteAddr())
		msg := dns.Msg{}
		msg.SetReply(r)
		for _, q := range r.Question {
			domain := strings.TrimSuffix(q.Name, ".")
			if record, found := records[domain]; found {
				var rr dns.RR
				var err error
				switch record.Type {
				case "A", "AAAA", "CNAME", "TXT", "NS", "PTR":
					rr, err = dns.NewRR(fmt.Sprintf("%s %s %s", q.Name, record.Type, record.Value))
				case "MX":
					rr, err = dns.NewRR(fmt.Sprintf("%s %s %d %s", q.Name, record.Type, record.Priority, record.Value))
				case "SRV":
					rr, err = dns.NewRR(fmt.Sprintf("%s %s %d %d %d %s", q.Name, record.Type, record.Priority, 0, 0, record.Value))
				default:
					log.Printf("Unsupported record type: %s", record.Type)
					continue
				}
				if err == nil {
					rr.Header().Ttl = record.TTL
					msg.Answer = append(msg.Answer, rr)
				} else {
					log.Printf("Failed to create RR: %v", err)
				}
			}
		}
		w.WriteMsg(&msg)
	}
}

func addGenericFlags(flagSets ...*flag.FlagSet) {
	for _, cmd := range flagSets {
		cmd.StringVar(&configPath, "config-path", defaultConfigPath, "Path to the config file")
	}
}

func printUsages(flagSets ...*flag.FlagSet) {
	for _, cmd := range flagSets {
		cmd.Usage()
	}
}

func main() {

	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	saveConfig := configCmd.Bool("save", false, "Save config template in ~/.easydns/config.json (change dir with -config-path flag)")
	printConfig := configCmd.Bool("print", false, "Prints configuration to stdout")
	printDefault := configCmd.Bool("template", false, "Instead of printing the current configuration, print the sample configuration")

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	addGenericFlags(configCmd, runCmd)

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s [config|run]\n\n\n", "easydns")
		printUsages(configCmd, runCmd)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "config":
		configCmd.Parse(os.Args[2:])
		if *saveConfig {
			data, err := json.MarshalIndent(DefaultConfig, "", "  ")
			if err != nil {
				log.Fatalf("failed to marshal default config: %v", err)
			}
			err = os.WriteFile(configPath, data, 0644)
			if err != nil {
				log.Fatalf("failed to save default config: %v", err)
			}

			// Exit after saving the default config
		} else if *printConfig {
			if *printDefault {
				config = &DefaultConfig
			} else {
				config, err = LoadConfig(configPath)
				if err != nil {
					log.Fatalf("cannot print config because %v", err)
				}
			}
			data, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				log.Fatalf("failed to marshal default config: %v", err)
			}
			fmt.Println(string(data))
		} else {
			configCmd.Usage()
		}
		os.Exit(0)
	case "run":
		runCmd.Parse(os.Args[2:])
	default:
		break
	}

	config, err = LoadConfig(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dns.HandleFunc(".", handleDNSRequest(config.Records))

	server := &dns.Server{Addr: ":" + config.Port, Net: "udp"}
	log.Printf("starting DNS server on port %s", config.Port)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
