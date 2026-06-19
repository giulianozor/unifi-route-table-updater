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
		verbose = true
	}
	client, err := unifi.NewClient(cfg.UnifiBaseURL, cfg.UnifiAPIKey, cfg.UnifiSite, cfg.Insecure, cfg.CACert)
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	client.Verbose = cfg.Verbose

	if *listSitesFlag {
		sites, err := client.ListSites()
		if err != nil {
			log.Fatalf("list sites: %v", err)
		}
		var pretty bytes.Buffer
		json.Indent(&pretty, []byte(sites), "", "  ")
		fmt.Println(pretty.String())
		return
	}

	if *getRouteFlag {
		route, err := client.GetStaticRoute(cfg.RouteID)
		if err != nil {
			log.Fatalf("get route: %v", err)
		}
		b, _ := json.MarshalIndent(route, "", "  ")
		fmt.Println(string(b))
		return
	}

	if *infoFlag {
		info, err := client.GetInfo()
		if err != nil {
			log.Fatalf("info: %v", err)
		}
		var pretty bytes.Buffer
		json.Indent(&pretty, []byte(info), "", "  ")
		fmt.Println(pretty.String())
		return
	}

	if *listRoutesFlag {
		routes, err := client.ListStaticRoutes()
		if err != nil {
			log.Fatalf("list routes: %v", err)
		}
		b, _ := json.MarshalIndent(routes, "", "  ")
		fmt.Println(string(b))
		return
	}

	if *dryRunFlag {
		log.Println("DRY RUN — no routes will be modified")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	run(cfg, client, *dryRunFlag)
	if *onceFlag {
		return
	}
	for {
		select {
		case <-ticker.C:
			run(cfg, client, *dryRunFlag)
		case <-sigCh:
			verbosef("shutting down")
			return
		}
	}
}

func run(cfg *config.Config, client *unifi.Client, dryRun bool) {
	verbosef("resolving DNS name...")
	ip, err := dns.ResolveIPv4(cfg.DNSName)
	if err != nil {
		log.Fatalf("dns resolution failed: %v", err)
	}
	verbosef("resolved %s -> %s", cfg.DNSName, ip)

	newDest := ip + cfg.RouteCIDR

	verbosef("checking current static route...")
	route, err := client.GetStaticRoute(cfg.RouteID)
	if err != nil {
		log.Fatalf("failed to get route: %v", err)
	}
	verbosef("current route destination: %s", route.Destination)

	if route.Destination == newDest {
		verbosef("route is already up-to-date")
		return
	}

	verbosef("destination changed from %s to %s", route.Destination, newDest)

	if dryRun {
		log.Printf("DRY RUN: would update route %s (%s) destination to %s", route.Name, route.ID, newDest)
		return
	}

	verbosef("updating...")
	route.Destination = newDest
	if err := client.UpdateStaticRoute(route); err != nil {
		log.Fatalf("failed to update route: %v", err)
	}
	fmt.Printf("SUCCESS: updated route %s (%s) destination to %s\n", route.Name, route.ID, newDest)
}
