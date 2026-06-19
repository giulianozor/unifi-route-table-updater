package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/giuliano/urt/internal/config"
	"github.com/giuliano/urt/internal/dns"
	"github.com/giuliano/urt/internal/telegram"
	"github.com/giuliano/urt/internal/unifi"
)

var verbose bool

func verbosef(format string, v ...any) {
	if verbose {
		log.Printf(format, v...)
	}
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	dryRunFlag := flag.Bool("dry-run", false, "log what would change without updating the route")
	onceFlag := flag.Bool("once", false, "run once and exit")
	listRoutesFlag := flag.Bool("list-routes", false, "list all static routes and exit")
	insecureFlag := flag.Bool("insecure", false, "skip TLS certificate verification")
	verboseFlag := flag.Bool("verbose", false, "enable verbose debug output")
	infoFlag := flag.Bool("info", false, "query the UniFi integration API for controller info and exit")
	listSitesFlag := flag.Bool("list-sites", false, "list all sites and exit")
	getRouteFlag := flag.Bool("get-route", false, "get the current route configuration as JSON and exit")
	testTelegramFlag := flag.Bool("test-telegram", false, "send a test telegram message and exit")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if *insecureFlag {
		cfg.Insecure = true
	}
	if *verboseFlag {
		cfg.Verbose = true
	}
	verbose = cfg.Verbose

	if *testTelegramFlag {
		if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
			log.Fatalf("telegram_bot_token and telegram_chat_id must be set in config")
		}
		msg := "This is a test message from urt"
		if err := telegram.Send(cfg.TelegramBotToken, cfg.TelegramChatID, msg); err != nil {
			log.Fatalf("telegram test failed: %v", err)
		}
		fmt.Println("test telegram message sent")
		return
	}

	client, err := unifi.NewClient(cfg.UnifiBaseURL, cfg.UnifiAPIKey, cfg.UnifiSite, cfg.Insecure, cfg.CACert)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	client.Verbose = cfg.Verbose

	if *listSitesFlag {
		body, err := client.ListSites()
		if err != nil {
			log.Fatalf("list sites: %v", err)
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(body), "", "  "); err != nil {
			fmt.Println(body)
		} else {
			fmt.Println(pretty.String())
		}
		return
	}

	if *getRouteFlag {
		body, err := client.GetStaticRouteRaw(cfg.RouteID)
		if err != nil {
			log.Fatalf("get route: %v", err)
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(body), "", "  "); err != nil {
			fmt.Println(body)
		} else {
			fmt.Println(pretty.String())
		}
		return
	}

	if *infoFlag {
		body, err := client.GetInfo()
		if err != nil {
			log.Fatalf("info: %v", err)
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(body), "", "  "); err != nil {
			fmt.Println(body)
		} else {
			fmt.Println(pretty.String())
		}
		return
	}

	if *listRoutesFlag {
		body, err := client.ListStaticRoutesRaw()
		if err != nil {
			log.Fatalf("list routes: %v", err)
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(body), "", "  "); err != nil {
			fmt.Println(body)
		} else {
			fmt.Println(pretty.String())
		}
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	if err := run(cfg, client, *dryRunFlag); err != nil {
		if *onceFlag {
			os.Exit(1)
		}
	}
	if *onceFlag {
		return
	}
	for {
		select {
		case <-ticker.C:
			_ = run(cfg, client, *dryRunFlag)
		case <-sigCh:
			verbosef("shutting down")
			return
		}
	}
}

func run(cfg *config.Config, client *unifi.Client, dryRun bool) error {
	verbosef("resolving DNS name...")
	ip, err := dns.ResolveIPv4(cfg.DNSName)
	if err != nil {
		log.Printf("dns resolution failed: %v", err)
		return err
	}
	verbosef("resolved %s -> %s", cfg.DNSName, ip)

	newDest := ip + cfg.RouteCIDR

	if dryRun {
		log.Printf("DRY RUN — checking %s (would update if changed)", cfg.DNSName)
	}

	verbosef("checking current static route...")
	route, err := client.GetStaticRoute(cfg.RouteID)
	if err != nil {
		log.Printf("failed to get route: %v", err)
		return err
	}
	verbosef("current route destination: %s", route.Destination)

	if route.Destination == newDest {
		if dryRun {
			log.Printf("DRY RUN: route is already up-to-date (%s)", route.Destination)
		} else {
			verbosef("route is already up-to-date")
		}
		return nil
	}

	if dryRun {
		log.Printf("DRY RUN: would update route %s (%s) destination to %s", route.Name, route.ID, newDest)
		return nil
	}

	oldDest := route.Destination
	verbosef("destination changed from %s to %s", oldDest, newDest)

	route.Destination = newDest
	if err := client.UpdateStaticRoute(route); err != nil {
		log.Printf("failed to update route: %v", err)
		return err
	}
	fmt.Printf("SUCCESS: updated route %s (%s) destination to %s\n", route.Name, route.ID, newDest)

	if cfg.TelegramEnabled {
		msg := fmt.Sprintf("Route %s updated: %s -> %s", route.Name, oldDest, newDest)
		if err := telegram.Send(cfg.TelegramBotToken, cfg.TelegramChatID, msg); err != nil {
			log.Printf("telegram notification failed: %v", err)
		}
	}
	return nil
}
