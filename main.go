package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/rdegges/go-ipify"
	"github.com/robfig/cron/v3"

	"github.com/teelyjc/home/internal/logger"
)

type Config struct {
	Token   string `json:"token" yaml:"token"`
	Domains []struct {
		Domain string `json:"domain" yaml:"domain"`
		Name   string `json:"name" yaml:"name"`
	} `json:"domains" yaml:"domains"`
}

type Usecases struct {
	client *cloudflare.Client
	config *Config
}

func new(config *Config) *Usecases {
	return &Usecases{
		config: config,
		client: cloudflare.NewClient(
			option.WithAPIToken(config.Token),
		),
	}
}

func (u *Usecases) FindByDomainName(name string) (*zones.Zone, error) {
	zap.S().Info("finding a matches domain..")
	ctx := context.Background()
	response, err := u.client.Zones.List(ctx, zones.ZoneListParams{})
	if err != nil {
		zap.S().Error("error finding matches domain", zap.Error(err))
		return nil, err
	}

	for _, zone := range response.Result {
		if zone.Name == name {
			zap.S().Info("found matches domain as a result.")
			return &zone, nil
		}
	}

	return nil, nil
}

func (u *Usecases) FindBySubDomain(zone *zones.Zone, matchesSubDomain string) (*dns.RecordResponse, error) {
	zap.S().Info("finding a matches sub-domain..")
	ctx := context.Background()
	res, err := u.client.DNS.Records.List(ctx, dns.RecordListParams{
		ZoneID: cloudflare.String(zone.ID),
	})
	if err != nil {
		return nil, err
	}

	for _, record := range res.Result {
		if strings.Contains(record.Name, matchesSubDomain) {
			zap.S().Info("found matches sub-domain as a result")
			return &record, nil
		}
	}

	return nil, nil
}

func (u *Usecases) UpdateParams(zone *zones.Zone, record *dns.RecordResponse) error {
	zap.S().Info("getting a current public ip address..")
	ip, err := ipify.GetIp()
	if err != nil {
		return err
	}

	ctx := context.Background()
	body := &dns.ARecordParam{
		Name:    cloudflare.String(record.Name),
		TTL:     cloudflare.F(dns.TTL1),
		Type:    cloudflare.F(dns.ARecordTypeA),
		Comment: cloudflare.String("updated from home-update-ip"),
		Content: cloudflare.String(ip),
		Proxied: cloudflare.Bool(false),
	}

	zap.S().Info("updating a ip-address", zap.String("record", record.Name), zap.String("ip", ip))
	_, err = u.client.DNS.Records.Update(ctx, record.ID, dns.RecordUpdateParams{
		ZoneID: cloudflare.String(zone.ID),
		Body:   body,
	})

	if err != nil {
		return err
	}

	return nil
}

func LoadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func update(cfg *Config, u *Usecases) error {
	for _, domain := range cfg.Domains {
		zone, err := u.FindByDomainName(domain.Domain)
		if err != nil {
			return err
		}
		record, err := u.FindBySubDomain(zone, domain.Name)
		if err != nil {
			zap.S().Error("error finding sub-domain", zap.Error(err))
			return err
		}

		u.UpdateParams(zone, record)
	}

	return nil
}

func main() {
	logger.SetupLogger()
	defer zap.S().Sync()

	var configPath string
	flag.StringVar(&configPath, "config", "./config.yaml", "path to a config file")
	flag.Parse()

	zap.S().Info("loading a configuration")
	cfg, err := LoadConfig(configPath)
	if err != nil {
		zap.S().Fatal("error loads a config from file", zap.Error(err))
	}
	zap.S().Info("successfully load a configuration from file")

	s := new(cfg)

	zap.S().Info("job is running, every 15 mins")

	cron := cron.New()
	cron.AddFunc("*/15 * * * * ", func() {
		update(cfg, s)
	})

	go cron.Start()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-signals

	zap.S().Info("running clean-up tasks..")
	cron.Stop()
}
